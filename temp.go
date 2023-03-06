package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

var dummyError = errors.New("dummy error")

func main() {
	i, _ := strconv.Atoi(os.Args[1])
	fmt.Println(i)
	fmt.Println(do(i))
}

func do(i int) error {
	if err := action1(i); err != nil {
		return fmt.Errorf("failed action1: %w", err)
	}
	if err := action2(i); err != nil {
		return fmt.Errorf("failed action2: %w", err)
	}
	return nil
}

func action1(i int) error {
	if err := action1_1(i); err != nil {
		return fmt.Errorf("failed action1_1: %w", err)
	}
	if err := action1_2(i); err != nil {
		return fmt.Errorf("failed action1_2: %w", err)
	}
	return nil
}

func action1_1(i int) error {
	if i == 1 {
		return dummyError
	}
	return nil
}

func action1_2(i int) error {
	if i == 2 {
		return dummyError
	}
	return nil
}

func action2(i int) error {
	if err := action2_1(i); err != nil {
		return fmt.Errorf("failed action2_1: %w", err)
	}
	if err := action2_2(i); err != nil {
		return fmt.Errorf("failed action2_2: %w", err)
	}
	return nil
}

func action2_1(i int) error {
	if i == 3 {
		return dummyError
	}
	return nil
}

func action2_2(i int) error {
	if i == 4 {
		return dummyError
	}
	return nil
}
