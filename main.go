package main

import (
	"fmt"

	"github.com/webitel/im-account-service/cmd"

	_ "github.com/webitel/im-account-service/cmd/migrate"
	_ "github.com/webitel/im-account-service/cmd/server"
)

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Println(err.Error())
		return
	}
}
