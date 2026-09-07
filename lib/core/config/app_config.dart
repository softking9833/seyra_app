enum AppEnvironment {
  development,
  staging,
  production,
}

/// Runtime backend configuration. Secrets must not be stored here.
///
/// Set at build time:
/// `--dart-define=SEYRA_ENV=production`
/// `--dart-define=SEYRA_API_BASE_URL=https://api.example.invalid`
final class AppConfig {
  const AppConfig({
    required this.environment,
    required this.apiBaseUrl,
  });

  final AppEnvironment environment;
  final String apiBaseUrl;

  static const defaultDevelopmentBaseUrl = 'https://127.0.0.1:8443';

  factory AppConfig.fromEnvironment() {
    const envName = String.fromEnvironment(
      'SEYRA_ENV',
      defaultValue: 'development',
    );
    const baseUrl = String.fromEnvironment(
      'SEYRA_API_BASE_URL',
      defaultValue: defaultDevelopmentBaseUrl,
    );

    return AppConfig(
      environment: _parseEnvironment(envName),
      apiBaseUrl: baseUrl,
    );
  }

  static AppEnvironment _parseEnvironment(String name) {
    return AppEnvironment.values.firstWhere(
      (value) => value.name == name,
      orElse: () => AppEnvironment.development,
    );
  }

  /// Staging and production must use HTTPS. Development should too.
  void validate() {
    final uri = Uri.tryParse(apiBaseUrl);
    if (uri == null || uri.host.isEmpty || !uri.hasScheme) {
      throw StateError('Invalid API base URL.');
    }

    if (environment != AppEnvironment.development && uri.scheme != 'https') {
      throw StateError(
        'TLS is required for the ${environment.name} environment.',
      );
    }
  }
}
