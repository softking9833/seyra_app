/// Domain-level failure. Keep this independent of Flutter and HTTP types.
sealed class Failure {
  const Failure(this.message);

  final String message;
}

final class UnexpectedFailure extends Failure {
  const UnexpectedFailure([super.message = 'Unexpected error']);
}
