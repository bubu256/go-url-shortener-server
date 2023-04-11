package shortener

import (
	"fmt"
	"sync"
	"time"
)

func ExampleCounterID_Run() {
	// создаем новый экземпляр счетчика
	counter := NewCounter(100)

	// запускаем горутину инкрементации
	counter.Run()

	// выдаем несколько ID
	for i := 0; i < 5; i++ {
		id := counter.Next()
		fmt.Println("Выдан ID:", id)
	}

	// ожидаем некоторое время
	time.Sleep(time.Second)

	// выдаем еще несколько ID
	for i := 0; i < 3; i++ {
		id := counter.Next()
		fmt.Println("Выдан ID:", id)
	}

	// Output:
	// Выдан ID: 101
	// Выдан ID: 102
	// Выдан ID: 103
	// Выдан ID: 104
	// Выдан ID: 105
	// Выдан ID: 106
	// Выдан ID: 107
	// Выдан ID: 108
}

func ExampleCounterID_Run_goroutine() {
	// создаем новый экземпляр счетчика
	counter := NewCounter(100)

	// запускаем горутину инкрементации
	counter.Run()

	wg := sync.WaitGroup{}
	// запрашиваем Next ID из разных конкурирующих потоков
	// при этом ожидаем что все полученные ID будут уникальные и находится в диапазоне 101 - 108
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Выдан ID:", counter.Next())
		}()
	}
	// ожидаем завершения всех горутин
	wg.Wait()
}
