package hash_circle

type ConsistentHash interface {
	Add(node string)
	Remove(node string)
	Gets(targetNode string, arr []string) []string
	GetAll(arr []string) map[string][]string
	Members() []string
}
