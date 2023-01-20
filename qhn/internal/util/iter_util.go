package util

func ChunkSlice[T any](slice []T, chunkSize uint) <-chan []T {
	sliceChan := make(chan []T)
	sliceLen := uint(len(slice))

	go func() {
		defer close(sliceChan)
		for idx := uint(0); idx < sliceLen; idx += chunkSize {
			if idx+chunkSize > sliceLen {
				sliceChan <- slice[idx:sliceLen]
			} else {
				sliceChan <- slice[idx : idx+chunkSize]
			}
		}
	}()

	return sliceChan
}
