package main

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import "unsafe"

//export godwit_plugin_init
func godwit_plugin_init(hostABI C.uint32_t, api *C.godwit_plugin_api) C.int {
	if hostABI != C.GODWIT_PLUGIN_ABI || api == nil {
		return -1
	}
	C.dlx_set_api(api)
	return 0
}

//export dlx_call
func dlx_call(method *C.char, req *C.uint8_t, size C.size_t, out *C.godwit_buffer) (rc C.int) {
	if method == nil || out == nil || size > 1<<20 || (size > 0 && req == nil) {
		return -1
	}
	out.ptr, out.len = nil, 0
	defer func() {
		if recover() != nil {
			rc = copyResponse(envelope(nil, failure("internal", "Plugin call failed.", false)), out)
		}
	}()
	return copyResponse(dispatch(C.GoString(method), C.GoBytes(unsafe.Pointer(req), C.int(size))), out)
}

func copyResponse(data []byte, out *C.godwit_buffer) C.int {
	out.ptr = C.CBytes(data)
	if out.ptr == nil {
		return -1
	}
	out.len = C.size_t(len(data))
	return 0
}

func main() {}
