package util

import (
	"io/fs"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
)

var NamespacedName = types.NamespacedName{Name: "some-domain.dedyn.io", Namespace: "desec-dns-operator"}

func CreateConfigDir(t *testing.T, serverUrl string) string {
	dir := t.TempDir()
	os.Mkdir(dir+"/config", fs.ModePerm)
	os.Mkdir(dir+"/secret", fs.ModePerm)

	runtime.Must(os.WriteFile(dir+"/config/domain", []byte("some-domain.dedyn.io"), fs.ModePerm))
	runtime.Must(os.WriteFile(dir+"/config/namespace", []byte("desec-dns-operator"), fs.ModePerm))
	runtime.Must(os.WriteFile(dir+"/config/mgmtHost", []byte(serverUrl), fs.ModePerm))
	runtime.Must(os.WriteFile(dir+"/config/updateIpHost", []byte(serverUrl), fs.ModePerm))
	runtime.Must(os.WriteFile(dir+"/secret/token", []byte("I'm a token"), fs.ModePerm))

	return dir
}
