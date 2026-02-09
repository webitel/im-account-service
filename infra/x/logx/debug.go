package logx

import (
	"os"
	"slices"
	"strings"
)

// $WBTL_LOG_DEBUG=[module][,module]...
var debugMode any // bool, []string

func init() {

  vs := os.Getenv("WBTL_LOG_DEBUG")
  // if all, e := strconv.ParseBool(vs); e == nil {
  //   debug = all // bool
  //   return
  // }
  vs = strings.ToLower(vs)
  switch vs {
	case "1", "on", "yes", "true", "all":
		debugMode = true
    return
	case "", "0", "off", "no", "false", "none":
		debugMode = false
    return
	}
  // strings.ToLower(!) above ..
  debugMode = strings.Split(vs, ",")
}

func isDebugModule(name string) bool {
  if name == "" {
    return false
  }
  modules, ok := debugMode.([]string)
  if !ok || len(modules) == 0 {
    return false
  }
  name = strings.ToLower(name)
  return slices.Contains(modules, name)
}

func Debug(module string, alias ...string) bool {
  if full, is := debugMode.(bool); is {
    return full
  }
  if isDebugModule(module) {
    return true
  }
  for _, a := range alias {
    if isDebugModule(a) {
      return true
    }
  }
  return false
}