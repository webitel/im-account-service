package logx

import (
	"os"
	"slices"
	"strings"
)

// $WBTL_LOG_DEBUG=[module][,module]...
var debugMode any // bool, []string

func init() {

  debugMode = false // default: disabled
  vs := os.Getenv("WBTL_LOG_DEBUG")
  // if all, e := strconv.ParseBool(vs); e == nil {
  //   debug = all // bool
  //   return
  // }
  vs = strings.ToLower(vs)
  modules := strings.Split(vs, ",")
  for i, n := 0, len(modules); i < n; i++ {
    switch modules[i] {
    case "1", "on", "yes", "true", "all":
      debugMode = true // ALL
      return
    case "0", "off", "no", "false", "none":
      // debugMode = false // default: false
      return
    case "":
      modules = append(modules[:i], modules[i+1:]...)
      n--
      i--
    } 
  }
  if len(modules) > 0 {
    // debugMode.([]string) ; set of module name(s) to enable debug for ..
    debugMode = modules
  }
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