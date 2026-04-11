package pearcut

func Ptr[T any](v T) *T { return &v }
