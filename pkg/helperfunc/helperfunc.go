// модуль содержит вспомогательные функции общие для нескольких модулей
package helperfunc

import "sync"

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
