package storage

type PG struct {
	Repository
}

func NewStoragePG(uri string) (*PG, error) {
	return &PG{}, nil
}
