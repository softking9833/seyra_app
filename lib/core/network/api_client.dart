/// Network transport boundary.
///
/// Protocol is not locked (HTTP, gRPC, or other).
/// Staging and production implementations must use TLS.
/// Do not call this from UI.
abstract interface class ApiClient {}
