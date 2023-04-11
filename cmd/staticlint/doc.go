/*
Package staticlint содержит исполняемый файл staticlint.exe реализующий утилиту командной строки для запуска multichecker

multichecker содержит след анализаторы:

  - ExitCheckAnalyzer - анализатор, который ищет прямые вызовы функции os.Exit в анализируемых файлах.

  - printf.Analyzer - анализатор, который ищет ошибки форматирования строк в функциях fmt.Printf и fmt.Sprintf.

  - shadow.Analyzer - анализатор, который ищет скрытие переменных во внутреннем блоке.

  - structtag.Analyzer - анализатор, который проверяет соответствие структурам определенных тегов.

  - staticcheck.Analyzers - набор анализаторов из библиотеки статического анализа golang.org/x/tools/go/analysis/staticcheck.

  - quickfix.Analyzers - набор анализаторов из библиотеки автоматического исправления кода honnef.co/go/tools/quickfix.

  - stylecheck.Analyzers - набор анализаторов из библиотеки стилевого анализа кода honnef.co/go/tools/stylecheck.

    Коды проверок анализаторов staticcheck.Analyzers, quickfix.Analyzers, stylecheck.Analyzers
    должны быть указаны в файле config.json.
    Допускается указание только первых двух символов, тогда будут запускаться все подходящие проверки.

Пример файла config.json:

	{
		"staticcheck": [
			"SA",
			"S1000",
			"ST1000",
			"QF1001"
		]
	}
*/
package staticlint
