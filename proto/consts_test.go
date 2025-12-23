package proto

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestWriteProto(t *testing.T) {
	src := []byte("hello world\n\n--benbens--\nt\n\n--benbens--\nt--benbens--\nt--benbens--\n")

	//proto := WriteProto(src, 1, "FromTo")
	//println(string(proto))
	proto := handleDataBody(src)
	println(string(proto))
	after := removeNewlineAfter(proto)
	if string(after) != string(src) {
		t.Errorf("after:%s \n, src:%s", string(after), string(src))
	}
}
func TestWriteProtoSpeed(t *testing.T) {
	for i := 0; i < 100000; i++ {
		TestWriteProto(t)
	}
}
func TestParseInt64ToBytes(t *testing.T) {
	var testNum int64 = 127
	fmt.Printf("%v", parseInt64ToBytes(testNum))
}

func TestPacket_Marshal(t *testing.T) {
	singlePacket := make([]byte, 145)
	file, err := os.OpenFile("D:\\go\\gogetway\\getwayServer\\log.txt", os.O_RDONLY, 0666)
	if err != nil {
		panic(err)
	}
	io.ReadFull(file, singlePacket)
	packet, err := UnMarshal(singlePacket)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", packet)
}

func TestReadProtoFromReader(t *testing.T) {
	file, err := os.OpenFile("D:\\go\\gogetway\\getwayServer\\log.txt", os.O_RDONLY, 0666)
	if err != nil {
		panic(err)
	}
	scanner, err := ReadProtoFromReader(file)
	for scanner.Scan() {
		bytes := scanner.Bytes()
		marshal, err := UnMarshal(bytes)
		if err != nil {
			t.Errorf("%v", err)
			return
		}
		data, err := json.Marshal(marshal)
		if err != nil {
			t.Errorf("%v", err)
			return
		}
		fmt.Printf("%s\n", string(data))
	}
}
