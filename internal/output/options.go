package output

type Options struct {
	Update bool
}

type Option func(*Options)

func Update(b bool) Option {
	return func(args *Options) {
		args.Update = b
	}
}
