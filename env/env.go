package env

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type EnvVars struct {
	Capacity *int
	Memory   *int
}

func LoadEnv() EnvVars {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, reading from environment variables.")
	}
	memory := os.Getenv("Memory")
	capacity := os.Getenv("Capacity")
	if memory == "" && capacity == "" {
		log.Fatalln("Please Provide Memory or Capacity in environment variable ")
	}
	var envs EnvVars
	if memory != "" {
		memoryInt, err := strconv.Atoi(memory)
		if err != nil {
			log.Fatalln("Invalid memory value: must be an integer")
		}
		envs.Memory = &memoryInt
	}
	if capacity != "" {
		capacityInt, err := strconv.Atoi(capacity)
		if err != nil {
			log.Fatalln("Invalid capacity value: must be an integer")
		}

		envs.Capacity = &capacityInt
	}
	return envs
}
