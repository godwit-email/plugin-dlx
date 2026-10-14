#include <stdlib.h>
#include "bridge.h"

extern int dlx_call(char *, uint8_t *, size_t, godwit_buffer *);

static int call(const char *method, const uint8_t *req, size_t len, godwit_buffer *out) {
    return dlx_call((char *)method, (uint8_t *)req, len, out);
}

static void release(void *ptr, size_t len) {
    (void)len;
    free(ptr);
}

void dlx_set_api(godwit_plugin_api *api) {
    api->abi_version = GODWIT_PLUGIN_ABI;
    api->call = call;
    api->free_buffer = release;
}
