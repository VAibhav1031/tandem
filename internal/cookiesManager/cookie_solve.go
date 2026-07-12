package cookies

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"

	_ "modernc.org/sqlite"
)

func copyfile(dest_name string, src string) (*os.File, error) {
	//******************COPYING_COOKIE_FILE**********

	// path of the Cookie file

	cookie_org, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer cookie_org.Close()

	temp_cookie_path, err := os.CreateTemp("", dest_name+"-*")
	//need to create the temp fiel and all shit
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(temp_cookie_path, cookie_org)

	return temp_cookie_path, err
}

func CookieSolver() string {

	// we need to open the  shit

	home_dir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("No HomeDir found!!", err)
	}

	cookie_path := home_dir + "/.config/chromium/Default/Cookies"
	temp_file_path_info, err := copyfile("Cookies", cookie_path)

	defer os.Remove(temp_file_path_info.Name())
	defer temp_file_path_info.Close()
	if err != nil {
		slog.Error("Creation of Temp Failed", err)
		return ""
	}

	db, err := sql.Open("sqlite", temp_file_path_info.Name())
	if err != nil {
		slog.Error("Error Occurred %v", err)
		return ""
	}
	defer db.Close()

	// Ping to the db to check if connection working well or not
	if err := db.Ping(); err != nil {
		fmt.Println("Failed to connect:", err)
	}

	// rows, err := db.Query("SELECT name FROM sqlite_master ;")
	// if err != nil {
	// 	slog.Error(err)
	// }
	// defer rows.Close()
	//
	// fmt.Println("Tables in database:")
	// for rows.Next() {
	// 	var tableName string
	// 	if err := rows.Scan(&tableName); err != nil {
	// 		slog.Error(err)
	// 	}
	// 	fmt.Printf("- %s\n", tableName)
	// }
	//
	// get the latest one
	rows, err := db.Query("Select encrypted_value from cookies where name = 'cf_clearance' and host_key = '.testfile.org' order by creation_utc desc limit 1;")

	if err != nil {
		slog.Error("Query Failed over the cookie Table", err)
	}
	defer rows.Close()

	var encrypted_value []byte
	for rows.Next() {
		if err := rows.Scan(&encrypted_value); err != nil {
			slog.Error("Unable to Scan the encrypted Value", err)
		}
		//fmt.Printf("- %s\n", row_)
	}
	// slog.Error("raw bytes length: %d", len(encrypted_value))
	// slog.Error("first 6 bytes: %x", encrypted_value[:6])
	// slog.Error("prefix string: %s", string(encrypted_value[:3]))
	//
	key := DbusKeyGetter()
	cookie, err := decryptCookie(encrypted_value, key)
	if err != nil {
		slog.Error("Error In Decryption: ", err)
		return ""
	}
	return cookie
}
