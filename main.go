package main

import (
	db "GoStore/database"
	l "GoStore/log"
	server "GoStore/routes"
)

func main() {
	l.LogMessage(l.INFO, "----> \033[1mStarting Server\033[0m <----")
	db.InitDB()
	server.StartServer()

}
