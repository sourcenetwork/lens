// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

/*!
This crate contains members to aid in the construction of a Rust [Lens Module](https://github.com/lens-vm/spec#abi---wasm-module-functions).

To get started, see [define](macro.define.html).
*/

use std::mem;
use std::mem::ManuallyDrop;
use std::iter::Iterator;
use std::marker::PhantomData;
use std::io::{Cursor, Write};
use serde::{Serialize, Deserialize};
use byteorder::{ReadBytesExt, WriteBytesExt, LittleEndian};

/// [Result](https://doc.rust-lang.org/std/result/enum.Result.html) type alias returned by lens_sdk.
pub mod result;

/// Error types returned by lens_sdk.
pub mod error;

/// Option type returned by lens_sdk.
pub mod option;

/// Alias for an sdk [Error](error/enum.Error.html).
pub type Error = error::Error;

/// Alias for an sdk [Result](result/type.Result.html).
pub type Result<T> = result::Result<T>;

/// Alias for an sdk [StreamOption](option/enum.StreamOption.html).
pub type StreamOption<T> = option::StreamOption<T>;

/// A type id that denotes a simple string-based error.
///
/// If present at the beginning of a byte array being read by a [lens host](https://github.com/sourcenetwork/lens#Hosts)
/// or [try_from_mem](fn.try_from_mem.html), the byte array will be treated as an error and will be
/// handled accordingly.
pub const ERROR_TYPE_ID: i8 = -1;

/// A type id that denotes a nil value.
pub const NIL_TYPE_ID: i8 = 0;

/// A type id that denotes a json value.
///
/// If present at the beginning of a byte array being read by a [lens host](https://github.com/sourcenetwork/lens#Hosts)
/// or [try_from_mem](fn.try_from_mem.html), the byte array will be treated as a json value and will be
/// handled accordingly.
pub const JSON_TYPE_ID: i8 = 1;

/// A type id that donates the end of stream.
///
/// If recieved it signals that the end of the stream has been reached and that the source will no longer yield
/// new values.
pub const EOS_TYPE_ID: i8 = i8::MAX;

/// Returns a nil pointer.
///
/// The pointer points to a zeroed byte, and will be interpretted by a [lens host](https://github.com/sourcenetwork/lens#Hosts)
/// as a nil value.
pub fn nil_ptr() -> *mut u8 {
    &mut 0u8
}

/// Allocate the given `size` number of bytes in memory and returns a pointer to
/// the first byte.
///
/// The runtime will be instructed to manually drop this memory and not dispose of it now - the value
/// that is to be held at this location should be written before any other calls are made into
/// the wasm instance.
pub fn alloc(size: usize) -> *mut u8 {
    let buf = Vec::with_capacity(size);
    let mut buf = ManuallyDrop::new(buf);
    let ptr = buf.as_mut_ptr();
    return ptr;
}

/// Manually drop the memory of the given size at the given location.
///
/// It should only be called on pointers to manually managed memory.
pub unsafe fn free(ptr: *mut u8, size: usize) {
    let buf: Vec<u8> = unsafe {
        Vec::from_raw_parts(ptr, size, size)
    };
    let buf = ManuallyDrop::new(buf);
    ManuallyDrop::into_inner(buf);
}

/// Manually drop the memory occupied by a transport buffer at the given location.
///
/// Items are transported across the host-wasm boundary as manually managed memory buffers, the
/// allocation for which is handled by [alloc](fn.alloc.html).  The pointer to this manually managed memory
/// is returned by the `next` host function imported by LensVM wasm modules.
///
/// Once the item has been consumed by the transform, the memory allocated must be manually dropped,
/// typically via this function.
///
/// # Safety
///
/// This function assumes that the pointer points to a transport buffer, passing a pointer to anything else
/// will result in undefined behaviour.
///
/// The pointer should not be used after calling this function.  The memory it points to will have been unreserved,
/// and may have been replaced by other stuff by the runtime.
///
/// # Errors
///
/// This function will return an [Error](error/enum.Error.html) if the data at the given location is not in the expected
/// format.
pub unsafe fn free_transport_buffer(ptr: *mut u8) -> Result<()> {
    let type_vec: Vec<u8> = unsafe {
        Vec::from_raw_parts(ptr, mem::size_of::<i8>(), mem::size_of::<i8>())
    };

    let type_rdr = Cursor::new(type_vec);
    let mut type_rdr = ManuallyDrop::new(type_rdr);

    let type_id: i8  = type_rdr.read_i8()?;
    if type_id == NIL_TYPE_ID {
        ManuallyDrop::into_inner(type_rdr);
        return Ok(())
    }
    if type_id == EOS_TYPE_ID {
        ManuallyDrop::into_inner(type_rdr);
        return Ok(())
    }
    if type_id < 0 {
        ManuallyDrop::into_inner(type_rdr);
        return Ok(())
    }

    let len_vec: Vec<u8> = unsafe {
        Vec::from_raw_parts(ptr.add(mem::size_of::<i8>()), mem::size_of::<u32>(), mem::size_of::<u32>())
    };

    let len_rdr = Cursor::new(len_vec);
    let mut len_rdr = ManuallyDrop::new(len_rdr);

    let len: usize = len_rdr.read_u32::<LittleEndian>()?.try_into()?;

    unsafe {
        free(ptr, mem::size_of::<i8>()+mem::size_of::<u32>()+len);
    }

    Ok(())
}

/// Read the data held at the given location in memory as the given `TOutput` type.
///
/// The bytes at the given location are expected to be in the correct format for the first (`type_id`) byte.
///
/// `type_id` | Type | Expected format
/// --- | --- | ---
/// < 0 | error | N/A - unsupported, will return an [InputErrorUnsupportedError](error/enum.LensError.html#variant.InputErrorUnsupportedError)
/// 0 | null value | N/A - will return [None](https://doc.rust-lang.org/std/option/enum.Option.html#variant.None)
/// \> 0 | JSON | \[`len`\]\[`json_string`\] where len is the length of bytes in the json_string
///
/// # Safety
///
/// The pointer given to this function will typically be result of calls to the `next` host function imported by LensVM wasm modules.  It is
/// manually managed memory, and calling this function will free the memory located at the given `ptr` - it must not be used after this call.
///
/// # Errors
///
/// This function will return an [Error](error/enum.Error.html) if the data at the given location is not in the expected
/// format.
pub unsafe fn try_from_mem<TOutput: for<'a> Deserialize<'a>>(ptr: *mut u8) -> Result<StreamOption<TOutput>> {
    let type_vec: Vec<u8> = unsafe {
        Vec::from_raw_parts(ptr, mem::size_of::<i8>(), mem::size_of::<i8>())
    };

    let mut type_rdr = Cursor::new(type_vec.clone());
    let _ = ManuallyDrop::new(type_vec);

    let type_id: i8  = type_rdr.read_i8()?;
    if type_id == NIL_TYPE_ID {
        unsafe {
            free_transport_buffer(ptr)?;
        }
        return Ok(StreamOption::None)
    }
    if type_id == EOS_TYPE_ID {
        unsafe {
            free_transport_buffer(ptr)?;
        }
        return Ok(StreamOption::EndOfStream)
    }
    if type_id < 0 {
        unsafe {
            free_transport_buffer(ptr)?;
        }
        return Result::from(error::LensError::InputErrorUnsupportedError)
    }

    let len_vec: Vec<u8> = unsafe {
        Vec::from_raw_parts(ptr.add(mem::size_of::<i8>()), mem::size_of::<u32>(), mem::size_of::<u32>())
    };

    let mut len_rdr = Cursor::new(len_vec.clone());
    let _ = ManuallyDrop::new(len_vec);

    let len: usize = len_rdr.read_u32::<LittleEndian>()?.try_into()?;

    let input_vec: Vec<u8> = unsafe {
        Vec::from_raw_parts(ptr.add(mem::size_of::<i8>()+mem::size_of::<u32>()), len, len)
    };

    // Clone the json bytes from the transport buffer, allowing subsequent code to operate on safely managed memory. 
    let json_bytes = input_vec.clone();
    let _ = ManuallyDrop::new(input_vec);

    // Now the transport pointer has been fully consumed and copied into managed memory, we can free the entire manually
    // managed transport buffer.
    unsafe {
        free_transport_buffer(ptr)?;
    }

    // It is possible for null json values to reach this line, particularly if sourced directly
    // from a 3rd party module, so we ensure that we parse to option as well as the earlier type_id
    // checks.
    let result = match serde_json::from_slice::<Option<TOutput>>(&json_bytes)? {
        Some(v) => StreamOption::Some(v),
        None => StreamOption::None,
    };

    Ok(result)
}

/// Write the given `message` bytes to memory, returning a pointer to the first byte.
///
/// Bytes are written in the same format as expected by [lens hosts](https://github.com/sourcenetwork/lens#Hosts) and
/// [try_from_mem](fn.try_from_mem.html) \- `[type_id][len][json_string]`.
///
/// # Errors
///
/// This function may return the same errors that [io::write](https://doc.rust-lang.org/std/io/trait.Write.html#tymethod.write)
/// may return.
pub fn try_to_mem(type_id: i8, message: &[u8]) -> Result<*mut u8> {
    let buffer = Vec::with_capacity(message.len() + mem::size_of::<u32>() + mem::size_of::<i8>());
    let mut wtr = Cursor::new(buffer);

    wtr.write_i8(type_id)?;
    wtr.set_position(mem::size_of::<i8>() as u64);

    match type_id {
        EOS_TYPE_ID => (),
        _ => {
            wtr.write_u32::<LittleEndian>(message.len() as u32)?;
            wtr.set_position(mem::size_of::<i8>() as u64 + mem::size_of::<u32>() as u64);
            wtr.write(message)?;
        }
    };

    let mut buffer = wtr.into_inner().clone();
    let result = buffer.as_mut_ptr();
    Ok(result)
}

/// Write the given `message` bytes to memory, returning a pointer to the first byte.
///
/// Bytes are written in the same format as expected by [lens hosts](https://github.com/sourcenetwork/lens#Hosts) and
/// [try_from_mem](fn.try_from_mem.html) \- `[type_id][len][json_string]`.
///
/// This function wraps [try_to_mem](fn.try_to_mem.html), if an error was generated by that internal function call
/// this function will attempt to write the error to memory, if that generates an error, it will attempt to write
/// a [FailedToWriteErrorToMemError](error/enum.LensError.html#variant.FailedToWriteErrorToMemError) to memory,
/// if that fails this function will panic.
///
/// # Panics
///
/// This function will panic if an error was generated internally and all attempts to write it to memory failed.
pub fn to_mem(type_id: i8, message: &[u8]) -> *mut u8 {
    match try_to_mem(type_id, message) {
        Ok(v) => v,
        Err(e) => {
            match try_to_mem(ERROR_TYPE_ID, &e.to_string().as_bytes()) {
                Ok(v) => v,
                Err(_) => {
                    // We have to panic here if this fails, as we can't keep trying and failing to send the error
                    try_to_mem(ERROR_TYPE_ID, error::LensError::FailedToWriteErrorToMemError.to_string().as_bytes()).unwrap()
                }
            }
        }
    }
}

/// Execute the given `transform` once, returning a pointer to the (serialized for transport) output.
///
/// The returned pointer can be sent across the wasm boundary to the executing lens-engine.
///
/// The given `transform` will be given an iterator that yields items of type `TInput`, `transform` may iterate
/// through zero-to-many items before returning.
///
/// # Errors
///
/// The iterator given to `transform` may yield a [StreamOption](option/enum.StreamOption.html) if `next` returns
/// and invalid pointer. Implmentors of `transform`s may handle that however they choose, although panicing is discouraged.
///
/// `transform` is free to return any [Error] kinds that they like.  [next] will serialize any returned errors, returning a
/// pointer to the serialized content which may then be passed across the wasm boundary and handled by the executing lens engine.
///
/// # Examples
///
/// The following example lens contains a simple filter, only yielding inputs where the `type_name` property equals `"pass"`.
///
/// ```
/// # use std::error::Error;
/// # use std::iter::Iterator;
/// # use serde::{Serialize, Deserialize};
/// # use lens_sdk::StreamOption;
/// #
/// # #[link(wasm_import_module = "lens")]
/// # unsafe extern "C" {
/// #     fn next() -> *mut u8;
/// # }
/// #
/// # #[derive(Serialize, Deserialize)]
/// # pub struct Value {
/// #     pub name: String,
/// # 	  pub type_name: String,
/// # }
/// #
/// # #[unsafe(no_mangle)]
/// # pub extern "C" fn alloc(size: usize) -> *mut u8 {
/// #     lens_sdk::alloc(size)
/// # }
/// #
/// #[unsafe(no_mangle)]
/// pub extern "C" fn transform() -> *mut u8 {
///     lens_sdk::next(|| -> *mut u8 { unsafe { next() } }, try_transform)
/// }
///
/// fn try_transform(
///     iter: &mut dyn Iterator<Item = lens_sdk::Result<Option<Value>>>,
/// ) -> Result<StreamOption<Value>, Box<dyn Error>> {
///     for item in iter {
///         let input = match item? {
///             Some(v) => v,
///             None => continue,
///         };
///
///         if input.type_name == "pass" {
///             return Ok(StreamOption::Some(input))
///         }
///     }
///
///     Ok(StreamOption::EndOfStream)
/// }
/// ```
pub fn next<TInput: for<'a> Deserialize<'a>, TOutput: Serialize>(
    next: impl Fn() -> *mut u8,
    transform: impl Fn(
        // `transform` will only ever recieve lens_sdk errors, the lens should decide what to do with them
        &mut dyn Iterator<Item = Result<Option<TInput>>>,
    // `transform` must be permitted to return any kind of error that lens-authors decide to
    ) -> std::result::Result<StreamOption<TOutput>, Box<dyn std::error::Error>>,
) -> *mut u8 {
    let mut iterator = InputIterator::new(&next);

    match transform(&mut iterator) {
        Ok(o) => match o {
            StreamOption::Some(value) => match serde_json::to_vec(&value) {
                Ok(json) => to_mem(JSON_TYPE_ID, &json),
                Err(e) => to_mem(ERROR_TYPE_ID, &e.to_string().as_bytes()),
            },
            StreamOption::None => nil_ptr(),
            StreamOption::EndOfStream => to_mem(EOS_TYPE_ID, &[]),
        },
        Err(e) => to_mem(ERROR_TYPE_ID, &e.to_string().as_bytes())
    }
}

struct InputIterator<'a, TInput> {
    next_ptr: &'a dyn Fn() -> *mut u8,
    // `value_type` is used by the compiler in order to limit the number
    // of compiled types, PhantomData is the standard zero-cost way of doing this.
    value_type: PhantomData<TInput>,
}

impl<'a, TInput> InputIterator<'a, TInput> {
    fn new(next: &'a dyn Fn() -> *mut u8) -> InputIterator<'a, TInput> {
        InputIterator::<'a, TInput> {
            next_ptr: next,
            value_type: PhantomData,
        }
    }
}

impl<TInput> Iterator for InputIterator<'_, TInput>
    where TInput : for<'a> Deserialize<'a> {
    type Item = Result<Option<TInput>>;

    fn next(&mut self) -> Option<Self::Item> {
        let ptr = (self.next_ptr)();

        match unsafe{ try_from_mem::<TInput>(ptr) } {
            Ok(val) => match val {
                StreamOption::None => Some(Ok(None)),
                StreamOption::Some(v) => Some(Ok(Some(v))),
                // EndOfStream gets mapped to None, to match the Iterator interface
                StreamOption::EndOfStream => None,
            },
            Err(e) => Some(Err(e)),
        }
    }
}

/// Define the mandatory `alloc` function for this Lens.
///
/// It is responsible for allocating memory for input items and will be called by the Lens engine.
#[macro_export]
macro_rules! define_alloc {
    () => {
        #[unsafe(no_mangle)]
        pub extern "C" fn alloc(size: usize) -> *mut u8 {
            $crate::alloc(size)
        }
    };
}

/// Define the mandatory `next` function for this Lens.
///
/// It is responsible for pulling the pointer to the next input item from the Lens engine.
#[macro_export]
macro_rules! define_next {
    () => {
        #[link(wasm_import_module = "lens")]
        unsafe extern "C" {
            fn next() -> *mut u8;
        }
    };
}

/// Define the mandatory `transform` function for this Lens.
///
/// This macro wraps the provided `try_transform` function, providing the boilerplate required to handle input items
/// sent across the WASM boundary from the Lens engine. The resultant function is responsible for transforming
/// input items pulled in by [next()](macro.define_next.html) and yields a pointer to the serialized result.
///
/// It assumes that a `next()` function exists within the calling scope.
#[macro_export]
macro_rules! define_transform {
    ($try_transform:ident) => {
        #[unsafe(no_mangle)]
        pub extern "C" fn transform() -> *mut u8 {
            $crate::next(|| -> *mut u8 { unsafe { next() } }, $try_transform)
        }
    };
    ($next:ident, $try_transform:ident) => {
        #[unsafe(no_mangle)]
        pub extern "C" fn transform() -> *mut u8 {
            $crate::next($next(), $try_transform)
        }
    };
}

/// Define the optional `inverse` function for this Lens.
///
/// This macro wraps the provided `try_inverse` function, providing the boilerplate required to handle input items
/// sent across the WASM boundary from the Lens engine. The resultant function is responsible for inversing transforms
/// on input items pulled in by [next()](macro.define_next.html) and yields a pointer to the serialized result.
///
/// It assumes that a `next()` function exists within the calling scope.
#[macro_export]
macro_rules! define_inverse {
    ($try_inverse:ident) => {
        #[unsafe(no_mangle)]
        pub extern "C" fn inverse() -> *mut u8 {
            $crate::next(|| -> *mut u8 { unsafe { next() } }, $try_inverse)
        }
    };
    ($next:ident, $try_inverse:ident) => {
        #[unsafe(no_mangle)]
        pub extern "C" fn inverse() -> *mut u8 {
            $crate::next($next(), $try_inverse)
        }
    };
}

/// Define the optional `set_param` function for this Lens.
///
/// `set_param` is used to recieve static parameters from the Lens engine. If parameters are provided to the Lens
/// engine, this function will be called once on initialization, before items are are fed through the transform/inverse
/// functions.  This macro defines the boiler plate required to recieve them.
///
/// It takes the name of the variable in which the parameter should be stored, and the type of the parameter.
///
/// # Examples
///
/// ```
/// # use std::sync::RwLock;
/// # use serde::Deserialize;
/// #
/// #[derive(Deserialize, Clone)]
/// pub struct Parameters {
///     pub src: String,
///     pub dst: String,
/// }
///
/// static PARAMETERS: RwLock<Option<Parameters>> = RwLock::new(None);
///
/// lens_sdk::define_set_param!(PARAMETERS: Parameters);
/// ```
#[macro_export]
macro_rules! define_set_param {
    ($var:ident: $Type:ty) => {
        #[unsafe(no_mangle)]
        pub extern "C" fn set_param(ptr: *mut u8) -> *mut u8 {
            match try_set_param(ptr) {
                Ok(_) => $crate::nil_ptr(),
                Err(e) => $crate::to_mem($crate::ERROR_TYPE_ID, &e.to_string().as_bytes())
            }
        }

        fn try_set_param(ptr: *mut u8) -> Result<(), Box<dyn std::error::Error>> {
            let parameter =  unsafe { $crate::try_from_mem::<$Type>(ptr)? }
                .ok_or($crate::error::LensError::ParametersNotSetError)?;

            let mut dst = $var.write()?;
            *dst = Some(parameter);
            Ok(())
        }
    };
}

/// Define the boilerplate for the Lens.
///
/// This macro is the easiest way to setup your Lens file - provide this with a transform function and
/// you should have a valid Lens.
///
/// This macro handles multiple patterns, each handled pattern will generate a different set of functions.
///
/// - [next()](macro.define_next.html) and [alloc()](macro.define_alloc.html) will always be defined.
/// - [transform()](macro.define_transform.html) will be defined if given a single transform function.
/// - [transform()](macro.define_transform.html) and [inverse()](macro.define_inverse.html) will be
///   defined if given the identities of two transform/inverse functions.
/// - [set_param()](macro.define_set_param.html) will be defined if given the name of a variable to hold a param,
///   and its type.
///
/// # Examples
///
/// The following example contains a complete Lens that transforms inputs by incrementing input.age by 1.
///
/// ```
/// # use serde::{ Deserialize, Serialize };
/// # use lens_sdk::StreamOption;
/// # use std::error::Error;
/// #
/// lens_sdk::define!(try_transform);
///
/// #[derive(Serialize, Deserialize)]
/// pub struct Value {
///     pub name: String,
/// 	pub age: i64,
/// }
///
/// fn try_transform(
///     iter: &mut dyn Iterator<Item = lens_sdk::Result<Option<Value>>>,
/// ) -> Result<StreamOption<Value>, Box<dyn Error>> {
///     for item in iter {
///         let input = match item? {
///             Some(v) => v,
///             None => return Ok(StreamOption::None),
///         };
///
///         let result = Value {
///             name: input.name,
///             age: input.age + 1,
///         };
///
///         return Ok(StreamOption::Some(result))
///     }
///
///     Ok(StreamOption::EndOfStream)
/// }
/// ```
///
/// The next example contains a complete Lens that transforms, and inverses, the incrementing of input.age by the amount
/// provided by the configured paramter.
///
/// ```
/// # use std::sync::RwLock;
/// # use serde::{ Deserialize, Serialize };
/// # use lens_sdk::StreamOption;
/// # use lens_sdk::error::LensError;
/// # use std::error::Error;
/// #
/// lens_sdk::define!(PARAMETERS: Parameters, try_transform, try_inverse);
///
/// #[derive(Deserialize, Clone)]
/// pub struct Parameters {
///     pub magnitude: i64,
/// }
///
/// static PARAMETERS: RwLock<Option<Parameters>> = RwLock::new(None);
///
/// #[derive(Serialize, Deserialize)]
/// pub struct Value {
///     pub name: String,
/// 	pub age: i64,
/// }
///
/// fn try_transform(
///     iter: &mut dyn Iterator<Item = lens_sdk::Result<Option<Value>>>,
/// ) -> Result<StreamOption<Value>, Box<dyn Error>> {
///     let params = PARAMETERS.read()?
///         .clone()
///         .ok_or(LensError::ParametersNotSetError)?;
///
///     for item in iter {
///         let input = match item? {
///             Some(v) => v,
///             None => return Ok(StreamOption::None),
///         };
///
///         let result = Value {
///             name: input.name,
///             age: input.age + params.magnitude,
///         };
///
///         return Ok(StreamOption::Some(result))
///     }
///
///     Ok(StreamOption::EndOfStream)
/// }
///
/// fn try_inverse(
///     iter: &mut dyn Iterator<Item = lens_sdk::Result<Option<Value>>>,
/// ) -> Result<StreamOption<Value>, Box<dyn Error>> {
///     let params = PARAMETERS.read()?
///         .clone()
///         .ok_or(LensError::ParametersNotSetError)?;
///
///     for item in iter {
///         let input = match item? {
///             Some(v) => v,
///             None => return Ok(StreamOption::None),
///         };
///
///         let result = Value {
///             name: input.name,
///             age: input.age - params.magnitude,
///         };
///
///         return Ok(StreamOption::Some(result))
///     }
///
///     Ok(StreamOption::EndOfStream)
/// }
/// ```
#[macro_export]
macro_rules! define {
    () => {
        $crate::define_alloc!();
        $crate::define_next!();
    };
    ($try_transform:ident) => {
        $crate::define_alloc!();
        $crate::define_next!();
        $crate::define_transform!($try_transform);
    };
    ($try_transform:ident, $try_inverse:ident) => {
        $crate::define_alloc!();
        $crate::define_next!();
        $crate::define_transform!($try_transform);
        $crate::define_inverse!($try_inverse);
    };
    ($param_var:ident: $ParamType:ty) => {
        $crate::define_alloc!();
        $crate::define_next!();
        $crate::define_set_param!($param_var: $ParamType);
    };
    ($param_var:ident: $ParamType:ty, $try_transform:ident) => {
        $crate::define_alloc!();
        $crate::define_next!();
        $crate::define_set_param!($param_var: $ParamType);
        $crate::define_transform!($try_transform);
    };
    ($param_var:ident: $ParamType:ty, $try_transform:ident, $try_inverse:ident) => {
        $crate::define_alloc!();
        $crate::define_next!();
        $crate::define_set_param!($param_var: $ParamType);
        $crate::define_transform!($try_transform);
        $crate::define_inverse!($try_inverse);
    };
}
