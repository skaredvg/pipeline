package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	sizeBuffer        uint = 3
	timeOutReadBuffer uint = 500
)

// Кольцевой буфер
type RingBuffer struct {
	buf       []int
	sidx      int           // индекс элемента для добавления
	gidx      int           // индекс элемента для считывания
	buf_empty bool          // флаг пустого буфера
	mux       sync.Mutex    // для безопасного множественного доступа к буферу
	timeout   time.Duration // задержка вычитывания из буфера
}

func NewRingBuffer(size uint, timeout uint) *RingBuffer {
	return &RingBuffer{buf: make([]int, size),
		buf_empty: true,
		mux:       sync.Mutex{},
		timeout:   time.Duration(timeout) * time.Millisecond,
	}
}

// метод вставки элемента в буфер
func (r *RingBuffer) Set(n int) {
	fmt.Println("Set")
	r.mux.Lock()
	fmt.Println("Set begin: r.sidx=", r.sidx, " r.gidx=", r.gidx)
	fmt.Println("r.buf[r.sidx]=", r.buf[r.sidx], " замещен n=", n)
	r.buf[r.sidx] = n
	if r.gidx == r.sidx && !r.buf_empty {
		r.gidx++
	}

	if r.gidx >= len(r.buf) {
		r.gidx = 0
	}
	r.sidx++
	if r.sidx >= len(r.buf) {
		r.sidx = 0
	}
	r.buf_empty = false
	fmt.Println("Set end: r.sidx=", r.sidx, " r.gidx=", r.gidx)
	r.mux.Unlock()
}

// Метод считывания элемента из буфера
func (r *RingBuffer) Get() (int, bool) {
	if r.buf_empty {
		return 0, false
	}
	r.mux.Lock()
	e := r.buf[r.gidx]
	fmt.Println("Get begin: r.sidx=", r.sidx, " r.gidx=", r.gidx, " e=", e)
	r.gidx++
	if r.gidx >= len(r.buf) {
		r.gidx = 0
	}

	if r.gidx == r.sidx {
		r.buf_empty = true
	}
	fmt.Println("Get end: r.sidx=", r.sidx, " r.gidx=", r.gidx, " e=", e)
	r.mux.Unlock()
	return e, true
}

// заполяем кольцевой буфер из внешнего источника (канала)
func (r *RingBuffer) WriteToBuffer(ch <-chan int) <-chan any {
	done := make(chan any)
	go func(ch <-chan int) {
		defer close(done)
		for n := range ch {
			fmt.Println("WriteToBuffer Обрабатываю ", n)
			//<-time.After(r.timeout)
			r.Set(n)
		}
		fmt.Println("Вышел WriteToBuffer")
	}(ch)
	return done
}

// Читаем из кольцевого буфера во внешний приемник(канал)
func (r *RingBuffer) ReadFromBuffer(done <-chan any) <-chan int {
	c := make(chan int)
	go func() {
		defer func() {
			fmt.Println("Вышел ReadFromBuffer")
			close(c)
		}()
		for {
			<-time.After(r.timeout)
			//			fmt.Println("ReadFromBuffer перед чтением буфера!")
			select {
			case <-done:
				return
			default:
				if n, b := r.Get(); b {
					fmt.Println("ReadFromBuffer прочитал из буфера ", n, b)
					c <- n
					fmt.Println("ReadFromBuffer отправил в канал печати ", n)
				}
			}
		}

	}()
	return c
}

// фильтруем числа
func filterNumber(cs <-chan string) <-chan int {
	c := make(chan int)
	go func(cs <-chan string) {
		defer close(c)
		for s := range cs {
			i, err := strconv.Atoi(s)
			if err == nil {
				fmt.Println("filterNumber Обрабатываю ", i)
				c <- i
			}
		}
		fmt.Println("Вышел из filterNumber")
	}(cs)
	return c
}

// фильтруем положительные числа
func filterPositiveNumber(cn <-chan int) <-chan int {
	c := make(chan int)
	go func(cn <-chan int) {
		defer close(c)
		for n := range cn {
			if n >= 0 {
				fmt.Println("filterPositiveNumber Обрабатываю ", n)
				c <- n
			}
		}
		fmt.Println("Вышел из filterPositiveNumber")
	}(cn)
	return c
}

// фильтруем числа больше 0 и кратные 3
func filterDiv3(cn <-chan int) <-chan int {
	c := make(chan int)
	go func(cn <-chan int) {
		defer close(c)
		for n := range cn {
			if n > 0 && n%3 == 0 {
				fmt.Println("filterDiv3 Обрабатываю ", n)
				c <- n
			}
		}
		fmt.Println("Вышел из filterDiv3")
	}(cn)
	return c
}

// вывод в консоль инфорамции
func OutputToConsole(ch <-chan int) <-chan any {
	done := make(chan any)
	go func(ch <-chan int) {
		defer close(done)
		for n := range ch {
			fmt.Println("Получены данные ... ", n)
		}
		fmt.Println("Вышел OutputToConsole")
	}(ch)
	return done
}

// ввод данных из источника- консоли
func InputFromConsole() <-chan string {
	s := bufio.NewScanner(os.Stdin)
	c := make(chan string)
	fmt.Println("Вводите числа через пробел")
	fmt.Println("Строка \"end\" для завершения работы")
	go func() {
		defer close(c)
		for s.Scan() {
			t := s.Text()
			if t == "end" {
				break
			}
			ts := strings.Split(s.Text(), " ")
			for _, v := range ts {
				fmt.Println("InputFromConsole Обрабатываю ", v)
				c <- v
			}
		}
		fmt.Println("Вышел из InputFromConsole")
	}()
	return c
}

func Run(size uint, to uint) {
	buf := NewRingBuffer(size, to)
	c := filterDiv3(filterPositiveNumber(filterNumber(InputFromConsole())))
	done := buf.WriteToBuffer(c)
	c2 := buf.ReadFromBuffer(done)
	done2 := OutputToConsole(c2)
	<-done2
	fmt.Println("Вышел из Run")
}

func main() {
	Run(sizeBuffer, timeOutReadBuffer)
	fmt.Println("Вышел из main")
}
