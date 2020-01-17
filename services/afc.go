package services

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"iconsole/frames"
	"iconsole/tunnel"
	"io"
	"os"
	"path"
	"strconv"
	"time"
)

var (
	afcHeader = []byte{0x43, 0x46, 0x41, 0x36, 0x4C, 0x50, 0x41, 0x41}
)

const (
	AFCOperationInvalid              = 0x00000000 /* Invalid */
	AFCOperationStatus               = 0x00000001 /* Status */
	AFCOperationData                 = 0x00000002 /* Data */
	AFCOperationReadDir              = 0x00000003 /* ReadDir */
	AFCOperationReadFile             = 0x00000004 /* ReadFile */
	AFCOperationWriteFile            = 0x00000005 /* WriteFile */
	AFCOperationWritePart            = 0x00000006 /* WritePart */
	AFCOperationTruncateFile         = 0x00000007 /* TruncateFile */
	AFCOperationRemovePath           = 0x00000008 /* RemovePath */
	AFCOperationMakeDir              = 0x00000009 /* MakeDir */
	AFCOperationGetFileInfo          = 0x0000000A /* GetFileInfo */
	AFCOperationGetDeviceInfo        = 0x0000000B /* GetDeviceInfo */
	AFCOperationWriteFileAtomic      = 0x0000000C /* WriteFileAtomic (tmp file+rename) */
	AFCOperationFileOpen             = 0x0000000D /* FileRefOpen */
	AFCOperationFileOpenResult       = 0x0000000E /* FileRefOpenResult */
	AFCOperationFileRead             = 0x0000000F /* FileRefRead */
	AFCOperationFileWrite            = 0x00000010 /* FileRefWrite */
	AFCOperationFileSeek             = 0x00000011 /* FileRefSeek */
	AFCOperationFileTell             = 0x00000012 /* FileRefTell */
	AFCOperationFileTellResult       = 0x00000013 /* FileRefTellResult */
	AFCOperationFileClose            = 0x00000014 /* FileRefClose */
	AFCOperationFileSetSize          = 0x00000015 /* FileRefSetFileSize (ftruncate) */
	AFCOperationGetConnectionInfo    = 0x00000016 /* GetConnectionInfo */
	AFCOperationSetConnectionOptions = 0x00000017 /* SetConnectionOptions */
	AFCOperationRenamePath           = 0x00000018 /* RenamePath */
	AFCOperationSetFSBlockSize       = 0x00000019 /* SetFSBlockSize (0x800000) */
	AFCOperationSetSocketBlockSize   = 0x0000001A /* SetSocketBlockSize (0x800000) */
	AFCOperationFileRefLock          = 0x0000001B /* FileRefLock */
	AFCOperationMakeLink             = 0x0000001C /* MakeLink */
	AFCOperationGetFileHash          = 0x0000001D /* GetFileHash */
	AFCOperationSetFileModTime       = 0x0000001E /* SetModTime */
	AFCOperationGetFileHashRange     = 0x0000001F /* GetFileHashWithRange */
	/* iOS 6+ */
	AFCOperationFileSetImmutableHint             = 0x00000020 /* FileRefSetImmutableHint */
	AFCOperationGetSizeOfPathContents            = 0x00000021 /* GetSizeOfPathContents */
	AFCOperationRemovePathAndContents            = 0x00000022 /* RemovePathAndContents */
	AFCOperationDirectoryEnumeratorRefOpen       = 0x00000023 /* DirectoryEnumeratorRefOpen */
	AFCOperationDirectoryEnumeratorRefOpenResult = 0x00000024 /* DirectoryEnumeratorRefOpenResult */
	AFCOperationDirectoryEnumeratorRefRead       = 0x00000025 /* DirectoryEnumeratorRefRead */
	AFCOperationDirectoryEnumeratorRefClose      = 0x00000026 /* DirectoryEnumeratorRefClose */
	/* iOS 7+ */
	AFCOperationFileRefReadWithOffset  = 0x00000027 /* FileRefReadWithOffset */
	AFCOperationFileRefWriteWithOffset = 0x00000028 /* FileRefWriteWithOffset */
)

type AFCFileMode int

const (
	AFC_RDONLY   AFCFileMode = 0x00000001
	AFC_RW       AFCFileMode = 0x00000002
	AFC_WRONLY   AFCFileMode = 0x00000003
	AFC_WR       AFCFileMode = 0x00000004
	AFC_APPEND   AFCFileMode = 0x00000005
	AFC_RDAPPEND AFCFileMode = 0x00000006
)

const (
	AFCErrSuccess                = 0
	AFCErrUnknownError           = 1
	AFCErrOperationHeaderInvalid = 2
	AFCErrNoResources            = 3
	AFCErrReadError              = 4
	AFCErrWriteError             = 5
	AFCErrUnknownPacketType      = 6
	AFCErrInvalidArgument        = 7
	AFCErrObjectNotFound         = 8
	AFCErrObjectIsDir            = 9
	AFCErrPermDenied             = 10
	AFCErrServiceNotConnected    = 11
	AFCErrOperationTimeout       = 12
	AFCErrTooMuchData            = 13
	AFCErrEndOfData              = 14
	AFCErrOperationNotSupported  = 15
	AFCErrObjectExists           = 16
	AFCErrObjectBusy             = 17
	AFCErrNoSpaceLeft            = 18
	AFCErrOperationWouldBlock    = 19
	AFCErrIoError                = 20
	AFCErrOperationInterrupted   = 21
	AFCErrOperationInProgress    = 22
	AFCErrInternalError          = 23
	AFCErrMuxError               = 30
	AFCErrNoMemory               = 31
	AFCErrNotEnoughData          = 32
	AFCErrDirNotEmpty            = 33
)

type AFCLockType int

const (
	AFCLockSharedLock    AFCLockType = 1 | 4
	AFCLockExclusiveLock AFCLockType = 2 | 4
	AFCLockUnlock        AFCLockType = 8 | 4
)

type AFCLinkType int

const (
	AFCHardLink AFCLinkType = 1
	AFCSymLink  AFCLinkType = 2
)

func getCStr(strs ...string) []byte {
	b := &bytes.Buffer{}
	for _, v := range strs {
		b.WriteString(v)
		b.WriteByte(0)
	}
	return b.Bytes()
}

func getError(status uint64) error {
	switch status {
	case AFCErrUnknownError:
		return errors.New("UnknownError")
	case AFCErrOperationHeaderInvalid:
		return errors.New("OperationHeaderInvalid")
	case AFCErrNoResources:
		return errors.New("NoResources")
	case AFCErrReadError:
		return errors.New("ReadError")
	case AFCErrWriteError:
		return errors.New("WriteError")
	case AFCErrUnknownPacketType:
		return errors.New("UnknownPacketType")
	case AFCErrInvalidArgument:
		return errors.New("InvalidArgument")
	case AFCErrObjectNotFound:
		return errors.New("ObjectNotFound")
	case AFCErrObjectIsDir:
		return errors.New("ObjectIsDir")
	case AFCErrPermDenied:
		return errors.New("PermDenied")
	case AFCErrServiceNotConnected:
		return errors.New("ServiceNotConnected")
	case AFCErrOperationTimeout:
		return errors.New("OperationTimeout")
	case AFCErrTooMuchData:
		return errors.New("TooMuchData")
	case AFCErrEndOfData:
		return errors.New("EndOfData")
	case AFCErrOperationNotSupported:
		return errors.New("OperationNotSupported")
	case AFCErrObjectExists:
		return errors.New("ObjectExists")
	case AFCErrObjectBusy:
		return errors.New("ObjectBusy")
	case AFCErrNoSpaceLeft:
		return errors.New("NoSpaceLeft")
	case AFCErrOperationWouldBlock:
		return errors.New("OperationWouldBlock")
	case AFCErrIoError:
		return errors.New("IoError")
	case AFCErrOperationInterrupted:
		return errors.New("OperationInterrupted")
	case AFCErrOperationInProgress:
		return errors.New("OperationInProgress")
	case AFCErrInternalError:
		return errors.New("InternalError")
	case AFCErrMuxError:
		return errors.New("MuxError")
	case AFCErrNoMemory:
		return errors.New("NoMemory")
	case AFCErrNotEnoughData:
		return errors.New("NotEnoughData")
	case AFCErrDirNotEmpty:
		return errors.New("DirNotEmpty")
	}
	return nil
}

type AFCPacket struct {
	EntireLen uint64
	ThisLen   uint64
	PacketNum uint64
	Operation uint64
	Data      []byte
	Payload   []byte
}

func (this *AFCPacket) Map() map[string]string {
	m := make(map[string]string)
	strs := this.Array()
	if strs != nil {
		for i := 0; i < len(strs); i += 2 {
			m[strs[i]] = strs[i+1]
		}
	}
	return m
}

func (this *AFCPacket) Array() []string {
	if this.Operation == AFCOperationData {
		bs := bytes.Split(this.Payload, []byte{0})
		strs := make([]string, len(bs)-1)
		for i := 0; i < len(strs); i++ {
			strs[i] = string(bs[i])
		}
		return strs
	}
	return nil
}

func (this *AFCPacket) Uint64() uint64 {
	return binary.LittleEndian.Uint64(this.Data)
}

func (this *AFCPacket) Error() error {
	if this.Operation == AFCOperationStatus {
		status := this.Uint64()
		if status != AFCErrSuccess {
			return getError(status)
		}
	}
	return nil
}

type AFCService struct {
	service   *tunnel.Service
	packetNum uint64
}

func NewAFCService(device frames.Device) (*AFCService, error) {
	serv, err := startService(AFCServiceName, device)
	if err != nil {
		return nil, err
	}

	return &AFCService{service: serv}, nil
}

func (this *AFCService) send(operation uint64, data, payload []byte) error {
	this.packetNum++

	entireLen := uint64(40)

	if data != nil {
		entireLen += uint64(len(data))
	}

	if payload != nil {
		entireLen += uint64(len(payload))
	}

	tLen := uint64(40)
	if data != nil {
		tLen += uint64(len(data))
	}

	buf := bytes.NewBuffer([]byte{})
	buf.Write(afcHeader)
	if err := binary.Write(buf, binary.LittleEndian, entireLen); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, tLen); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, this.packetNum); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, operation); err != nil {
		return err
	}

	if data != nil {
		buf.Write(data)
	}

	sB := buf.Bytes()

	conn := this.service.GetConnection()

	sent := 0
	for sent < len(sB) {
		if n, err := conn.Write(sB[sent:]); err != nil {
			return err
		} else {
			sent += n
		}
	}

	sent = 0
	if payload != nil {
		for sent < len(payload) {
			if n, err := conn.Write(payload[sent:]); err != nil {
				return err
			} else {
				sent += n
			}
		}
	}

	return nil
}

func (this *AFCService) recv() (*AFCPacket, error) {
	offset := uint64(0x28)

	conn := this.service.GetConnection()

	header := make([]byte, offset)
	n, err := conn.Read(header)
	if err != nil && n == 0 {
		return nil, err
	}
	if n < 0x28 {
		return nil, errors.New("recv: header")
	}
	if bytes.Compare(header[:8], afcHeader) != 0 {
		return nil, errors.New("recv: header not match")
	}

	packet := &AFCPacket{}

	packet.EntireLen = binary.LittleEndian.Uint64(header[8:16])
	packet.ThisLen = binary.LittleEndian.Uint64(header[16:24])
	packet.PacketNum = binary.LittleEndian.Uint64(header[24:32])
	packet.Operation = binary.LittleEndian.Uint64(header[32:])

	buf := &bytes.Buffer{}
	pkgBuf := make([]byte, 0xffff)

	for offset < packet.EntireLen {
		if n, err := conn.Read(pkgBuf); err != nil && n <= 0 {
			return nil, err
		} else {
			buf.Write(pkgBuf[:n])
			offset += uint64(n)
		}
	}

	dataAndPayload := buf.Bytes()

	packet.Data = dataAndPayload[:int(packet.ThisLen-40)]
	packet.Payload = dataAndPayload[int(packet.ThisLen-40):]

	if err := packet.Error(); err != nil {
		return nil, err
	}

	return packet, nil
}

type AFCDeviceInfo struct {
	Model      string
	TotalBytes uint64
	FreeBytes  uint64
	BlockSize  uint64
}

func (this *AFCService) GetDeviceInfo() (*AFCDeviceInfo, error) {
	if err := this.send(AFCOperationGetDeviceInfo, nil, nil); err != nil {
		return nil, err
	}

	if b, err := this.recv(); err != nil {
		return nil, err
	} else {
		m := b.Map()
		totalBytes, err := strconv.ParseUint(m["FSTotalBytes"], 10, 64)
		if err != nil {
			return nil, err
		}
		freeBytes, err := strconv.ParseUint(m["FSFreeBytes"], 10, 64)
		if err != nil {
			return nil, err
		}
		blockSize, err := strconv.ParseUint(m["FSBlockSize"], 10, 64)
		if err != nil {
			return nil, err
		}

		return &AFCDeviceInfo{
			Model:      m["Model"],
			TotalBytes: totalBytes,
			FreeBytes:  freeBytes,
			BlockSize:  blockSize,
		}, nil
	}
}

func (this *AFCService) ReadDirectory(p string, prefix bool) ([]string, error) {
	if err := this.send(AFCOperationReadDir, getCStr(p), nil); err != nil {
		return nil, err
	}

	if b, err := this.recv(); err != nil {
		return nil, err
	} else {
		final := b.Array()[2:]

		if !prefix {
			return final, nil
		}

		var fix []string
		for _, v := range final {
			fix = append(fix, path.Join(p, v))
		}
		return fix, nil
	}
}

type afcFileInfo struct {
	name   string
	size   uint64
	mtime  uint64
	ifmt   string
	source map[string]string
}

func (this *afcFileInfo) Name() string {
	return this.name
}

/*
	for get physical size
	use st_blocks * (FSBlockSize / 8)
*/
func (this *afcFileInfo) Size() int64 {
	return int64(this.size)
}
func (this *afcFileInfo) Mode() os.FileMode {
	return os.ModeType
}
func (this *afcFileInfo) ModTime() time.Time {
	return time.Unix(0, int64(this.mtime))
}
func (this *afcFileInfo) IsDir() bool {
	return this.ifmt == "S_IFDIR"
}
func (this *afcFileInfo) Sys() interface{} {
	return this.source
}

func (this *AFCService) GetFileInfo(filename string) (os.FileInfo, error) {
	if err := this.send(AFCOperationGetFileInfo, getCStr(filename), nil); err != nil {
		return nil, err
	}

	if b, err := this.recv(); err != nil {
		return nil, err
	} else {
		m := b.Map()

		st_size, err := strconv.ParseUint(m["st_size"], 10, 64)
		if err != nil {
			return nil, err
		}
		st_mtime, err := strconv.ParseUint(m["st_mtime"], 10, 64)
		if err != nil {
			return nil, err
		}

		info := &afcFileInfo{
			name:   path.Base(filename),
			size:   st_size,
			mtime:  st_mtime,
			ifmt:   m["st_ifmt"],
			source: m,
		}

		return info, nil
	}
}

type AFCFile struct {
	service *AFCService
	fd      uint64
}

func (this *AFCService) FileOpen(filename string, filemode AFCFileMode) (*AFCFile, error) {
	b := getCStr(filename)
	buf := make([]byte, len(b)+8)
	copy(buf[8:], b)
	binary.LittleEndian.PutUint64(buf[:8], uint64(filemode))

	if err := this.send(AFCOperationFileOpen, buf, nil); err != nil {
		return nil, err
	}

	if b, err := this.recv(); err != nil {
		return nil, err
	} else if b.Operation == AFCOperationFileOpenResult {
		return &AFCFile{service: this, fd: b.Uint64()}, nil
	} else {
		return nil, fmt.Errorf("operation %d", b.Operation)
	}
}

func (this *AFCFile) op(o ...uint64) []byte {
	lo := 1
	if o != nil {
		lo = len(o) + 1
	}
	buf := make([]byte, lo*8)
	binary.LittleEndian.PutUint64(buf, this.fd)
	for i := 1; i < lo; i++ {
		binary.LittleEndian.PutUint64(buf[i*8:], o[i-1])
	}
	return buf
}

func (this *AFCFile) Lock(mode AFCLockType) error {
	if err := this.service.send(AFCOperationFileRefLock, this.op(uint64(mode)), nil); err != nil {
		return err
	}

	if b, err := this.service.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}

	return nil
}

func (this *AFCFile) Unlock() error {
	return this.Lock(AFCLockUnlock)
}

func (this *AFCFile) Read(p []byte) (int, error) {
	if err := this.service.send(AFCOperationFileRead, this.op(uint64(len(p))), nil); err != nil {
		return -1, err
	}

	if b, err := this.service.recv(); err != nil {
		return -1, err
	} else if err := b.Error(); err != nil {
		return -1, err
	} else {
		if b.Payload == nil {
			return 0, io.EOF
		}
		copy(p, b.Payload)
		return len(b.Payload), nil
	}
}

func (this *AFCFile) Write(p []byte) (int, error) {
	if err := this.service.send(AFCOperationFileWrite, this.op(), p); err != nil {
		return -1, err
	}

	if b, err := this.service.recv(); err != nil {
		return -1, err
	} else if err := b.Error(); err != nil {
		return -1, err
	} else {
		return len(p), nil
	}
}

func (this *AFCFile) Tell() (uint64, error) {
	if err := this.service.send(AFCOperationFileTell, this.op(), nil); err != nil {
		return 0, err
	} else if b, err := this.service.recv(); err != nil {
		return 0, err
	} else if err := b.Error(); err != nil {
		return 0, err
	} else if b.Operation == AFCOperationFileTellResult {
		return b.Uint64(), nil
	} else {
		return 0, fmt.Errorf("operation %d", b.Operation)
	}
}

func (this *AFCFile) Seek(offset int64, whence int) (int64, error) {
	if err := this.service.send(AFCOperationFileSeek, this.op(uint64(whence), uint64(offset)), nil); err != nil {
		return -1, err
	} else if b, err := this.service.recv(); err != nil {
		return -1, err
	} else if err := b.Error(); err != nil {
		return -1, err
	} else if t, err := this.Tell(); err != nil {
		return -1, err
	} else {
		return int64(t), nil
	}
}

func (this *AFCFile) Truncate(size int64) error {
	if err := this.service.send(AFCOperationFileSetSize, this.op(uint64(size)), nil); err != nil {
		return err
	} else if b, err := this.service.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}
	return nil
}

func (this *AFCFile) Close() error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, this.fd)
	if err := this.service.send(AFCOperationFileClose, b, nil); err != nil {
		return err
	} else if _, err := this.service.recv(); err != nil {
		return err
	}
	return nil
}

func (this *AFCService) Remove(path string) error {
	if err := this.send(AFCOperationRemovePath, getCStr(path), nil); err != nil {
		return err
	} else if b, err := this.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}
	return nil
}

func (this *AFCService) Rename(oldpath, newpath string) error {
	if err := this.send(AFCOperationRenamePath, getCStr(oldpath, newpath), nil); err != nil {
		return err
	} else if b, err := this.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}
	return nil
}

func (this *AFCService) Mkdir(path string) error {
	if err := this.send(AFCOperationMakeDir, getCStr(path), nil); err != nil {
		return err
	} else if b, err := this.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}
	return nil
}

func (this *AFCService) Link(linkType AFCLinkType, oldname, newname string) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(linkType))
	b = append(b, getCStr(oldname, newname)...)
	if err := this.send(AFCOperationMakeLink, b, nil); err != nil {
		return err
	} else if b, err := this.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}
	return nil
}

func (this *AFCService) Truncate(path string, newsize uint64) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, newsize)
	b = append(b, getCStr(path)...)
	if err := this.send(AFCOperationTruncateFile, b, nil); err != nil {
		return err
	} else if b, err := this.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}
	return nil
}

func (this *AFCService) SetFileTime(mtime uint64, path string) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, mtime)
	b = append(b, getCStr(path)...)
	if err := this.send(AFCOperationSetFileModTime, b, nil); err != nil {
		return err
	} else if b, err := this.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}
	return nil
}

/* sha1 algorithm */
func (this *AFCService) Hash(path string) ([]byte, error) {
	if err := this.send(AFCOperationGetFileHash, getCStr(path), nil); err != nil {
		return nil, err
	} else if b, err := this.recv(); err != nil {
		return nil, err
	} else if err := b.Error(); err != nil {
		return nil, err
	} else {
		return b.Payload, nil
	}
}

/* sha1 algorithm with file range */
func (this *AFCService) HashWithRange(start, end uint64, path string) ([]byte, error) {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint64(b, start)
	binary.LittleEndian.PutUint64(b[8:], end)
	b = append(b, getCStr(path)...)

	if err := this.send(AFCOperationGetFileHashRange, b, nil); err != nil {
		return nil, err
	} else if b, err := this.recv(); err != nil {
		return nil, err
	} else if err := b.Error(); err != nil {
		return nil, err
	} else {
		fmt.Printf("%x\n", b)
		return b.Payload, nil
	}
}

/* since iOS6+ */
func (this *AFCService) RemoveAll(path string) error {
	if err := this.send(AFCOperationRemovePathAndContents, getCStr(path), nil); err != nil {
		return err
	} else if b, err := this.recv(); err != nil {
		return err
	} else if err := b.Error(); err != nil {
		return err
	}
	return nil
}

func (this *AFCService) Close() error {
	return this.service.GetConnection().Close()
}
