package utils

const (
    // Reed-Solomon code with (n, k) parameters
    N int = 6
    K int = 3
    L int = 3
    StripeSize int = 4096 * 3 //bytes redis
    NumOfSlots int = 16384
)

var MasterNodes = [3]string {"localhost:50052", "localhost:50053", "localhost:50054"}
var Weights = [3]string {"100", "100", "100"}