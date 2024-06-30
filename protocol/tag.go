package protocol

func isTagListener(tag uint64) bool {
    return tag >> 63 == 0 
}

func isTagServer(tag uint64) bool {
    return tag >> 63 == 1
}

func getListenerTag(tag uint64) uint64 {
    mask := ^(uint64(1) << 63)
    return tag & mask
} 

