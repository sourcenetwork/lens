# Lens host-go

Lens/host-go contains a lens host implementation written in Go.

It contains two packages - `engine` is the core lens engine and allows programmatic usage of the lens engine.  `config` sits on top of `engine` and allows consumers to provide a lens file containing the configuration of multiple lenses that they wish to be applied to their source data.

Should persisted storage of lenses and their configuration be desirable, the `store` and `node` packages are available.  `node` is the higher level package and in addition to some handy configuration defaults will in the near future allow the sharing of lenses over a P2P network.
