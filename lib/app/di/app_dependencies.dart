/// Composition root.
///
/// Register implementations here when they exist. No DI package is used yet.
/// Core and features must not look up dependencies from widgets directly.
abstract final class AppDependencies {
  static Future<void> initialize() async {
    // Bindings will be added when concrete implementations exist.
  }
}
