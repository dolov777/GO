package main

import (
	"errors"
	"fmt"
	"math"
)

const IMTPower = 2

func main() {
	fmt.Println("__ Калькулятор индекса массы тела __")
	for {
		userKg, userHeight := gwtUserInput()
		IMT, err := calculateIMT(userKg, userHeight)
		if err != nil {
			fmt.Println("Не заданы параметры для расчёта")
			continue
			// panic("Не заданы параметры для расчта")
		}
		outputResult(IMT)
		isRepeateCalculation := checkRepeatCalculation()
		if !isRepeateCalculation {
			break
		}
	}
}

func outputResult(IMT float64) {
	result := fmt.Sprintf("Ваш индекс массы тела: %.0f\n", IMT)
	fmt.Print(result)
	switch {
	case IMT < 16:
		fmt.Println("У вас сильный дифицит массы тела")
	case IMT < 18.5:
		fmt.Println("У вас дифицит массы тела")
	case IMT < 25:
		fmt.Println("У вас нормальный вес")
	case IMT < 30:
		fmt.Println("У вас избыточный вес")
	default:
		fmt.Println("У вас степень ожирения")
	}
}

func calculateIMT(userKg float64, userHeight float64) (float64, error) {
	if userKg <= 0 || userHeight <= 0 {
		return 0, errors.New("NO_PARANS_ERROR")
	}
	IMT := userKg / math.Pow(userHeight/100, IMTPower)
	return IMT, nil
}

func gwtUserInput() (float64, float64) {
	var userHeight, userKg float64
	fmt.Print("Ведите свой рост в сантиметрах: ")
	fmt.Scan(&userHeight)
	fmt.Print("Ведите свой вес: ")
	fmt.Scan(&userKg)
	return userKg, userHeight
}

func checkRepeatCalculation() bool {
	var userChoise string
	fmt.Print("Вы хотите сделать ещё расчёт (y/n): ")
	fmt.Scan(&userChoise)
	if userChoise == "y" || userChoise == "Y" {
		return true
	}
	return false
}
