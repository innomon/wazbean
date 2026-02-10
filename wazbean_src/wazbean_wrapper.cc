#include <string>
#include <memory>

#include "beancount/cparser/parser.h"
#include "beancount/cparser/ledger.h"
#include "google/protobuf/util/json_util.h"

// C-style wrapper function to be exported to WebAssembly.
extern "C" {

// A single, static string to hold the JSON result.
// This is a simple approach to manage memory across the C++/WASM boundary.
// The string's memory is managed by the C++ runtime and will be valid
// until the next call to this function.
std::string result_json;

const char* parse_bql_to_json(const char* query_string) {
    if (query_string == nullptr) {
        result_json = "{\"error\": \"Input query was null.\"}";
        return result_json.c_str();
    }

    // Call the Beancount C++ parser.
    // The second argument is a dummy filename for error reporting.
    std::unique_ptr<beancount::Ledger> ledger = 
        beancount::parser::ParseString(query_string, "<wazbean>");

    if (!ledger) {
        result_json = "{\"error\": \"Parser returned a null ledger.\"}";
        return result_json.c_str();
    }

    // Configure JSON output options.
    google::protobuf::util::JsonPrintOptions json_options;
    json_options.add_whitespace = true;
    json_options.always_print_primitive_fields = true;

    // Serialize the resulting Ledger protobuf message to a JSON string.
    auto status = google::protobuf::util::MessageToJsonString(*ledger, &result_json, json_options);

    if (!status.ok()) {
        result_json = "{\"error\": \"Failed to serialize ledger to JSON.\"}";
        return result_json.c_str();
    }
    
    return result_json.c_str();
}

} // extern "C"