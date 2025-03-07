// package main

// import (
// 	"fmt"
// 	"sync"
// 	"time"
// )

// func try_me(start int, n int, ch chan int, wg *sync.WaitGroup) {
// 	defer wg.Done() // Notify the WaitGroup when the goroutine finishes

// 	// Correct loop to run 'n' times
// 	for i := 0; i < n; i++ {
// 		ch <- i + start
// 		time.Sleep(300 * time.Millisecond)
// 	}
// }

// func main() {
// 	fmt.Println("This is gooey")

// 	ch := make(chan int, 20) // Increase channel capacity to handle more data

// 	var wg sync.WaitGroup

// 	// Correct use of a simple loop to launch goroutines
// 	for i := 0; i < 4; i++ {
// 		wg.Add(1)
// 		go try_me(i*4, 5, ch, &wg)
// 	}

// 	// Close the channel after all goroutines have finished
// 	go func() {
// 		wg.Wait()
// 		close(ch)
// 	}()

// 	// Consume messages from channel
// 	for number := range ch {
// 		fmt.Println(number)
// 	}
// }
