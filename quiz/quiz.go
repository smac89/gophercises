package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

type QuizRecord struct {
	question string
	answer   string
	response string
}

const (
	defaultQuizFile = "problems.csv"
	defaultDuration = 30
	defaultShuffle  = false
)

func main() {
	var quizPath string
	var timeLimit int
	var shuffleQuiz bool

	flag.StringVar(&quizPath, "quiz", defaultQuizFile, "A csv file in the format of 'question,answer'")
	flag.IntVar(&timeLimit, "duration", defaultDuration, "A time limit for the quiz, in seconds")
	flag.BoolVar(&shuffleQuiz, "shuffle", defaultShuffle, "Shuffle the quiz questions?")
	flag.Parse()

	file, err := os.Open(quizPath)
	if err != nil {
		log.Printf("Failed to open file %s\n", quizPath)
		log.Fatal(err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer file.Close()
	records := readQuestions(file)

	if shuffleQuiz {
		shuffleQuestions(records)
	}

	fmt.Printf("Welcome to Quizbot. Please answer to the best of your knowledge\n")
	fmt.Printf("Press enter to start...\n")
	//goland:noinspection GoUnhandledErrorResult
	fmt.Scanln()

	var (
		correctResp, totalQuestions = 0, len(records)
	)

	for record := range startQuiz(records, time.Duration(timeLimit)*time.Second) {
		if record == nil {
			break
		}
		if record.response == record.answer {
			correctResp++
		}
	}

	fmt.Printf("\nThanks for taking the quiz. You scored %d/%d = %.1f%%\n", correctResp, totalQuestions, (float64)(correctResp)/(float64)(totalQuestions)*100)
}

func readQuestions(csvFile *os.File) []QuizRecord {
	var (
		records    []QuizRecord
		quizReader = csv.NewReader(csvFile)
	)
	quizReader.FieldsPerRecord = 2
	quizReader.TrimLeadingSpace = true
	quizReader.ReuseRecord = true

	for {
		record, err := quizReader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		question, answer := strings.TrimSpace(record[0]), strings.TrimSpace(record[1])
		records = append(records, QuizRecord{question: question, answer: answer})
	}

	return records
}

func shuffleQuestions(quizRecords []QuizRecord) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(quizRecords), func(i, j int) {
		quizRecords[i], quizRecords[j] = quizRecords[j], quizRecords[i]
	})
}

func startQuiz(quizRecords []QuizRecord, timelimit time.Duration) <-chan *QuizRecord {
	var (
		answerCh = make(chan string)
		respCh   = make(chan *QuizRecord)
		timer    = time.NewTimer(timelimit)
		response string
	)

	go func() {
		defer close(answerCh)

		for lineNum := range quizRecords {
			record := &quizRecords[lineNum]
			go func() {
				if _, err := fmt.Scanln(&response); err == nil {
					answerCh <- strings.TrimSpace(response)
				}
			}()
			select {
			case <-timer.C:
				fmt.Printf("\nTimeout!\n")
				respCh <- nil
				return
			case record.response = <-answerCh:
				respCh <- record
			}
		}
	}()

	return respCh
}
