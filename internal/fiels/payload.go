package fiels

type FindFileDto struct {
	fileName string `json:fileName`
	Path     string `json:path`
}
