package bitmap

type Bitmap struct {
	bits []byte
	size int
}

// [0,0,0,0,0,0,0,0] [0,0,0,0,0,0,0,0]
func NewBitmap(size int) *Bitmap {
	if size == 0 {
		size = 250
	}

	return &Bitmap{
		bits: make([]byte, size),
		size: size * 8, // 每个byte有8位
	}
}

func (b *Bitmap) Set(id string) {
	// id在哪个bit
	idx := hash(id) % b.size
	// 在哪个byte
	byteIdx := idx / 8
	// 在byte的哪个bit
	bitIdx := idx % 8

	// 设置对应的bit为1
	b.bits[byteIdx] |= (1 << bitIdx) // 1 << bitIdx表示将1左移bitIdx位（100...0），得到一个只有第bitIdx位为1的数，然后与当前byte进行按位或运算，设置对应的bit为1
}

func (b *Bitmap) IsSet(id string) bool {
	// id在哪个bit
	idx := hash(id) % b.size
	// 在哪个byte
	byteIdx := idx / 8
	// 在byte的哪个bit
	bitIdx := idx % 8
	// 判断对应的bit是否为1
	return (b.bits[byteIdx] & (1 << bitIdx)) != 0 // 1 << bitIdx表示将1左移bitIdx位（100...0），得到一个只有第bitIdx位为1的数，然后与当前byte进行按位与运算，如果结果不为0，说明对应的bit为1
}

func (b *Bitmap) Export() []byte {
	return b.bits
}

func Load(bits []byte) *Bitmap {
	if len(bits) == 0 {
		return NewBitmap(0)
	}

	return &Bitmap{
		bits: bits,
		size: len(bits) * 8,
	}
}

func hash(id string) int {
	// BKDRHash算法
	seed := 131313 // 31 131 1313 13131 131313 etc..
	hash := 0
	for _, c := range id {
		hash = hash*seed + int(c)
	}

	return hash & 0x7FFFFFFF // 保持为正数
}
