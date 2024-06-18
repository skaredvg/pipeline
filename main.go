package main

import (
	"bufio"
	"fmt"
	"log"
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
	f := "function Set:"
	r.mux.Lock()
	log.Printf("%s begin: r.sidx=%v  r.gidx=%v", f, r.sidx, r.gidx)
	log.Printf("%s r.buf[r.sidx]=%v замещен n=%v", f, r.buf[r.sidx], n)
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
	log.Println("%s end: r.sidx=%d r.gidx=%d", f, r.sidx, r.gidx)
	r.mux.Unlock()
}

// Метод считывания элемента из буфера
func (r *RingBuffer) Get() (int, bool) {
	f := "function Get:"
	if r.buf_empty {
		log.Printf("%s buf_empty=%v", f, r.buf_empty)
		return 0, false
	}
	r.mux.Lock()
	e := r.buf[r.gidx]
	log.Printf("%s begin: r.sidx=%d  r.gidx=%d e=%v", f, r.sidx, r.gidx, e)
	r.gidx++
	if r.gidx >= len(r.buf) {
		r.gidx = 0
	}

	if r.gidx == r.sidx {
		r.buf_empty = true
	}
	log.Printf("%s end: r.sidx=%d r.gidx=%d e=%v", f, r.sidx, r.gidx, e)
	r.mux.Unlock()
	return e, true
}

// заполяем кольцевой буфер из внешнего источника (канала)
func (r *RingBuffer) WriteToBuffer(ch <-chan int) <-chan any {
	f := "function WriteToBuffer"
	log.Printf("%s Begin", f)
	done := make(chan any)
	go func(ch <-chan int) {
		defer close(done)
		for n := range ch {
			log.Printf("%s executing %v", f, n)
			//<-time.After(r.timeout)
			r.Set(n)
		}
		log.Printf("%s End", f)
	}(ch)
	return done
}

// Читаем из кольцевого буфера во внешний приемник(канал)
func (r *RingBuffer) ReadFromBuffer(done <-chan any) <-chan int {
	f := "function ReadFromBuffer"
	log.Printf("%s Begin", f)
	c := make(chan int)
	go func() {
		defer func() {
			log.Printf("%s End", f)
			close(c)
		}()
		for {
			<-time.After(r.timeout)
			select {
			case <-done:
				return
			default:
				if n, b := r.Get(); b {
					log.Printf("%s n=%v b=%v", f, n, b)
					c <- n
					log.Printf("%s send to channel printing n=%v", f, n)
				}
			}
		}

	}()
	return c
}

// фильтруем числа
func filterNumber(cs <-chan string) <-chan int {
	f := "function filterNumber"
	log.Printf("%s Begin", f)
	c := make(chan int)
	go func(cs <-chan string) {
		defer close(c)
		for s := range cs {
			i, err := strconv.Atoi(s)
			if err == nil {
				log.Printf("%s executing i=%v", f, i)
				c <- i
			}
		}
		log.Printf("%s End", f)
	}(cs)
	return c
}

// фильтруем положительные числа
func filterPositiveNumber(cn <-chan int) <-chan int {
	f := "function filterPositiveNumber"
	c := make(chan int)
	log.Printf("%s Begin", f)
	go func(cn <-chan int) {
		defer close(c)
		for n := range cn {
			if n >= 0 {
				log.Printf("%s executing n=", f, n)
				c <- n
			}
		}
		log.Printf("%s End", f)
	}(cn)
	return c
}

// фильтруем числа больше 0 и кратные 3
func filterDiv3(cn <-chan int) <-chan int {
	f := "function filterDiv3"
	c := make(chan int)
	log.Printf("%s Begin", f)
	go func(cn <-chan int) {
		defer close(c)
		for n := range cn {
			if n > 0 && n%3 == 0 {
				log.Printf("%s excuting %v", f, n)
				c <- n
			}
		}
		log.Printf("%s End", f)
	}(cn)
	return c
}

// вывод в консоль инфорамции
func OutputToConsole(ch <-chan int) <-chan any {
	f := "OutputToConsole"
	done := make(chan any)
	log.Printf("%s Begin", f)
	go func(ch <-chan int) {
		defer close(done)
		for n := range ch {
			log.Printf("%s receiving data n=%v", f, n)
		}
		log.Printf("%s End", f)
	}(ch)
	return done
}

// ввод данных из источника- консоли
func InputFromConsole() <-chan string {
	f := "function InputFromConsole"
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
				log.Printf("%s excuting v=%v", f, v)
				c <- v
			}
		}
		log.Printf("%s End", f)
	}()
	return c
}

func Run(size uint, to uint) {
	f := "function Run"
	log.Printf("%s Begin", f)
	buf := NewRingBuffer(size, to)
	c := filterDiv3(filterPositiveNumber(filterNumber(InputFromConsole())))
	done := buf.WriteToBuffer(c)
	c2 := buf.ReadFromBuffer(done)
	done2 := OutputToConsole(c2)
	<-done2
	fmt.Println("%s End", f)
}

func main() {
	f := "function main"
	log.Printf("%s Begin", f)
	Run(sizeBuffer, timeOutReadBuffer)
	log.Printf("%s End", f)
}
