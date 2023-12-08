package buffer

// let's make default block size as 24 lines
const dataBlockSize = 24

type DataBlock struct {
	blockBuffer DataBuffer
}

type Partitioning []DataBlock

func MakePartitioning(results [][]string) (Partitioning, error) {
	var partitioning Partitioning

	var dataBlockCounter = 0
	buf := DataBuffer{}
	for idx, result := range results {
		// pass csv caption
		if idx == 0 {
			continue
		}
		err := buf.WriteLine(result)
		if err != nil {
			return nil, err
		}
		dataBlockCounter++

		if dataBlockCounter%dataBlockSize == 0 {
			dataBlock := partitioning.AddBlock(buf)
			partitioning = append(partitioning, *dataBlock)

			buf = DataBuffer{}
		}
	}

	return partitioning, nil
}

func (p *Partitioning) AddBlock(blockBuffer DataBuffer) *DataBlock {
	return &DataBlock{blockBuffer: blockBuffer}
}

func (b *DataBlock) Read() [][]string {
	res := b.blockBuffer.Next(dataBlockSize)
	b.blockBuffer.Reset()
	return res
}

func DefineBucketSize() int {
	/*
		Let's take a scenario where table size is: 1224 rows, our hardcoded block size: 24 rows.
		Now, divide 1224 / 24 = 51
		Now, remember number of bucket will always be in the power of 2.
		So we need to find n such that 2^n > 51, n = 6 {64}
		So, I am going to use number of buckets as 2^6 = 64
	*/
	return 64
}
