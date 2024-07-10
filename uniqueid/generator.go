package uniqueid

import (
	"crypto/cipher"
	"encoding/binary"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/xtea"
)

type Generator interface {
	NewID() ID
}

const (
	epoch             = 1577836800000 // 2020-01-01T00:00:00Z
	nodeBits          = 12
	instanceIndexBits = 6
	stepBits          = 10
	maxReservedId     = 0xffffffff
)

type generatorImpl struct {
	lock     sync.Mutex
	epoch    time.Time
	lastTime uint64
	nodeID   uint64
	step     uint64
	block    cipher.Block

	stepMax   uint64
	timeShift uint8
	nodeShift uint8
}

func NewGenerator(key []byte, nodeID int) (Generator, error) {
	nodeMax := -1 ^ (-1 << nodeBits)
	if nodeID < 0 || nodeID > nodeMax {
		return nil, errors.New("invalid_node_id")
	}
	c, err := xtea.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "new_xtea_cipher_error")
	}
	g := &generatorImpl{
		nodeID: uint64(nodeID),
		block:  c,
	}
	g.stepMax = -1 ^ (-1 << stepBits)
	g.timeShift = nodeBits + stepBits
	g.nodeShift = stepBits
	return g, nil
}

func NewGeneratorFromHostName(key []byte, serviceID int, debug bool) (Generator, error) {
	serviceIDMax := -1 ^ (-1 << (nodeBits - instanceIndexBits))
	if serviceID < 0 || serviceID > serviceIDMax {
		return nil, errors.New("service_id_overflow")
	}
	instanceIndex := 0
	if !debug {
		hostName, err := os.Hostname()
		if err != nil {
			return nil, errors.Wrap(err, "get_hostname_error")
		}
		lastIndex := strings.LastIndex(hostName, "-")
		if lastIndex > 0 {
			hostName = hostName[lastIndex+1:]
		}
		if hostName == "" {
			return nil, errors.New("invalid_hostname")
		}
		instanceIndex, err = strconv.Atoi(hostName)
		if err != nil {
			return nil, errors.Wrap(err, "parse_instance_index_error")
		}
		instanceIndexMax := -1 ^ (-1 << instanceIndexBits)
		if instanceIndex < 0 || instanceIndex > instanceIndexMax {
			return nil, errors.New("instance_index_overflow")
		}
	}
	nodeID := (serviceID << instanceIndexBits) | (instanceIndex)
	return NewGenerator(key, nodeID)
}

func (g *generatorImpl) NewID() ID {
	g.lock.Lock()
	defer g.lock.Unlock()
	for {
		plainID := g.snowflakeID()
		cipherBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(cipherBuf, plainID)
		g.block.Encrypt(cipherBuf, cipherBuf)
		cipherID := binary.BigEndian.Uint64(cipherBuf)
		if cipherID > maxReservedId {
			return ID(cipherID)
		}
	}
}

func (g *generatorImpl) snowflakeID() uint64 {
	now := uint64(time.Now().UnixMilli()) - epoch
	if now <= g.lastTime {
		g.step++
		if g.step > g.stepMax {
			g.lastTime++
			g.step = 0
		}
	} else {
		g.lastTime = now
		g.step = 0
	}
	return (g.lastTime << g.timeShift) | (g.nodeID << g.nodeShift) | (g.step)
}
