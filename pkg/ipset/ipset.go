package ipset

type IPSet interface {
	Name() string
	AddEntry(entry string) error
	DelEntry(entry string) error
	Count() int
}
