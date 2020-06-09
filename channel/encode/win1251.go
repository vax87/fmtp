package encode

import (
   "golang.org/x/text/encoding/charmap"
)

func Win1251toUtf8(str string) string {
   dec := charmap.Windows1251.NewDecoder()
   out, _ := dec.String(str)
   return out
}

func Utf8toWin1251(str string) string {
   enc := charmap.Windows1251.NewEncoder()
   out, _ := enc.String(str)
   return out
}