package storage

import (
	"errors"
	"strconv"
)

func IsOrderNumberValid(number string) error {

	n, err := strconv.Atoi(number)

	if err != nil {
		return err
	}

	if (n%10+checksum(n/10))%10 != 0 {
		return errors.New("Number is not valid!")
	}

	return nil
}

func checksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 {
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
