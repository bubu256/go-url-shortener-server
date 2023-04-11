// Пакет helperfunc содержит вспомогательные функции, которые используются несколькими модулями.
package helperfunc

import "sync"

// FanInSliceString - объединяет несколько каналов типа []string в один канал и возвращает его.
// Функция ожидает, что каждый канал будет закрыт после передачи всех данных.
func FanInSliceString(chs ...chan []string) chan []string {
	OutCh := make(chan []string)
	wg := &sync.WaitGroup{}
	go func() {
		for _, ch := range chs {
			wg.Add(1)
			go func(inCh <-chan []string) {
				defer wg.Done()
				for keyUser := range inCh {
					OutCh <- keyUser
				}
			}(ch)
		}
		wg.Wait()
		close(OutCh)
	}()
	return OutCh
}
