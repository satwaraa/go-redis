package env

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type EnvVars struct {
	Capacity  *int
	Tcp_port  *int
	Http_port *int
	Memory    *int
}

func LoadEnv() EnvVars {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, reading from environment variables.")
	}
	memory := os.Getenv("Memory")
	capacity := os.Getenv("CAPACITY")
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
	tcp_port := os.Getenv("TCP_PORT")
	if tcp_port == "" {
		log.Fatalln("Please Provide TCP_PORT in environment variable ")
	}
	tcp_portInt, err := strconv.Atoi(tcp_port)
	if err != nil {
		log.Fatalln("Invalid tcp_port value: must be an integer")
	}
	envs.Tcp_port = &tcp_portInt

	http_port := os.Getenv("HTTP_PORT")
	if http_port == "" {
		http_port = "8080" // default
	}
	http_portInt, err := strconv.Atoi(http_port)
	if err != nil {
		log.Fatalln("Invalid http_port value: must be an integer")
	}
	envs.Http_port = &http_portInt

	return envs
}
