//! logs_add_attribute_handle_rust is the Rust counterpart of
//! logs_add_attribute_handle: it adds a fixed attribute to every log record
//! through the handle/accessor ABI.
//!
//! Build with:
//!
//!     cargo build --release --target wasm32-wasip1

// Field IDs are OTLP protobuf field numbers, shared with the host side.
const FIELD_LOGS_RESOURCE_LOGS: u32 = 1;
const FIELD_RESOURCE_LOGS_SCOPE_LOGS: u32 = 2;
const FIELD_SCOPE_LOGS_LOG_RECORDS: u32 = 2;
const FIELD_LOG_RECORD_ATTRIBUTES: u32 = 6;

const TELEMETRY_TYPE_LOGS: u32 = 1 << 1;

#[link(wasm_import_module = "opentelemetry.io/wasm")]
extern "C" {
    #[link_name = "pdataCurrentLogs"]
    fn pdata_current_logs() -> u32;
    #[link_name = "pdataCommitCurrentLogs"]
    fn pdata_commit_current_logs();
    #[link_name = "pdataGetMsg"]
    fn pdata_get_msg(handle: u32, field: u32) -> u32;
    #[link_name = "pdataSliceLen"]
    fn pdata_slice_len(handle: u32) -> u32;
    #[link_name = "pdataSliceGet"]
    fn pdata_slice_get(handle: u32, index: u32) -> u32;
    #[link_name = "pdataMapPutStr"]
    fn pdata_map_put_str(handle: u32, key_ptr: u32, key_len: u32, val_ptr: u32, val_len: u32);
}

fn map_put_str(handle: u32, key: &str, value: &str) {
    unsafe {
        pdata_map_put_str(
            handle,
            key.as_ptr() as u32,
            key.len() as u32,
            value.as_ptr() as u32,
            value.len() as u32,
        );
    }
}

#[no_mangle]
pub extern "C" fn getSupportedTelemetry() -> u32 {
    TELEMETRY_TYPE_LOGS
}

#[no_mangle]
pub extern "C" fn processLogs() -> u32 {
    unsafe {
        let logs = pdata_current_logs();
        let rls = pdata_get_msg(logs, FIELD_LOGS_RESOURCE_LOGS);
        for i in 0..pdata_slice_len(rls) {
            let sls = pdata_get_msg(pdata_slice_get(rls, i), FIELD_RESOURCE_LOGS_SCOPE_LOGS);
            for j in 0..pdata_slice_len(sls) {
                let lrs = pdata_get_msg(pdata_slice_get(sls, j), FIELD_SCOPE_LOGS_LOG_RECORDS);
                for k in 0..pdata_slice_len(lrs) {
                    let attrs = pdata_get_msg(pdata_slice_get(lrs, k), FIELD_LOG_RECORD_ATTRIBUTES);
                    map_put_str(attrs, "example_key", "example_value");
                }
            }
        }
        pdata_commit_current_logs();
    }
    0
}
