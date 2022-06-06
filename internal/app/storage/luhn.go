package storage

// CalculateLuhn return the check number
func CalculateLuhn(number int64) int {
	checkNumber := checksum(number)

	if checkNumber == 0 {
		return 0
	}
	return 10 - checkNumber
}

// Valid check number is valid or not based on Luhn algorithm
func OrderIsValid(number int64) bool {
	return (int(number%10)+checksum(number/10))%10 == 0
}

func checksum(number int64) int {
	var luhn int64

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return int(luhn % 10)
}
