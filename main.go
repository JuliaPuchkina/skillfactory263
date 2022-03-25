package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// кольцевой буфер целых чисел
type RingIntBuffer struct {
	array []int
	pos   int        // текущая позиция кольцевого буфера
	size  int        // общий размер буфера
	m     sync.Mutex // мьютекс для потокобезопасного доступа к буферу
}

// создание нового буфера целых чисел
func NewRingIntBuffer(size int) *RingIntBuffer {
	return &RingIntBuffer{make([]int, size), -1, size, sync.Mutex{}}
}

// добавление нового элемента в конец буфера
func (r *RingIntBuffer) Push(el int) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.pos == r.size-1 {
		for i := 1; i <= r.size-1; i++ {
			r.array[i-1] = r.array[i]
		}
		r.array[r.pos] = el
	} else {
		r.pos++
		r.array[r.pos] = el
	}
}

// получение всех элементов буфера и его последующая очистка
func (r *RingIntBuffer) Get() []int {
	if r.pos < 0 {
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()
	var output []int = r.array[:r.pos+1]
	r.pos = -1
	return output
}

func writeToBuffer(currentChan <-chan int, r *RingIntBuffer) {
	for number := range currentChan {
		r.Push(number)
		log.Print("Number added to buffer:", number)
	}
}

func writeToConsole(r *RingIntBuffer, t *time.Ticker) {
	for range t.C {
		buffer := r.Get()
		if len(buffer) > 0 {
			fmt.Println("Получены следующие данные:", buffer)
			log.Print("Numbers received from buffer:", buffer)
		}
	}
}

func main() {

	// стадия фильтрации отрицательных чисел
	noNegative := func(done <-chan int, input <-chan int) <-chan int {
		noNegStream := make(chan int)
		go func() {
			defer close(noNegStream)
			for i := range input {
				if i >= 0 {
					select {
					case noNegStream <- i:
						log.Print("Number is not negative:", i)
					case <-done:
						return
					}
				} else {
					log.Print("Negative number filtered:", i)
				}
			}
		}()
		return noNegStream
	}

	// стадия фильтрации чисел, не кратных 3, и 0
	onlyThree := func(done <-chan int, input <-chan int) <-chan int {
		onlyThreeStream := make(chan int)
		go func() {
			defer close(onlyThreeStream)
			for i := range input {
				if i%3 == 0 {
					if i > 0 {
						select {
						case onlyThreeStream <- i:
							log.Print("Number is a multiple of 3 and doesn't equal 0:", i)
						case <-done:
							return
						}
					} else {
						log.Print("Number equals 0, filtered:", i)
					}
				} else {
					log.Print("Number is not a multiple of 3, filtered:", i)
				}
			}
		}()
		return onlyThreeStream
	}

	// канал для централизованной остановки конвейера
	done := make(chan int)
	defer close(done)

	// получение чисел из консоли
	fmt.Println("Введите числа:")

	// запуск конвейера
	input := make(chan int)
	go read(input)
	pipeline := onlyThree(done, noNegative(done, input))

	// буфер
	size := 10
	r := NewRingIntBuffer(size)
	go writeToBuffer(pipeline, r)

	// вывод данных
	delay := 5
	ticker := time.NewTicker(time.Second * time.Duration(delay))
	writeToConsole(r, ticker)
}

// функция чтения из консоли
func read(input chan<- int) {
	log.Print("Pipeline launched")
	for {
		var num int
		_, err := fmt.Scanf("%d\n", &num)
		if err != nil {
			fmt.Println("Введены неверные данные")
			log.Print("Wrong data type")
		} else {
			log.Print("Nuber received:", num)
			input <- num
		}
	}
}

