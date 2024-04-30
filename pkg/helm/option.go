package helm

type Options struct {
	Verbose    bool
	Update     bool
	K8SVersion string
}

type Option func(*Options)

func Verbose(b bool) Option {
	return func(args *Options) {
		args.Verbose = b
	}
}

func Update(b bool) Option {
	return func(args *Options) {
		args.Update = b
	}
}

func K8SVersion(v string) Option {
	return func(args *Options) {
		args.K8SVersion = v
	}
}
