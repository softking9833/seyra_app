import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/features/auth/domain/entities/auth_session.dart';
import 'package:seyra/features/auth/domain/failures/auth_failures.dart';
import 'package:seyra/features/auth/domain/repositories/auth_repository.dart';
import 'package:seyra/features/auth/domain/usecases/login_use_case.dart';
import 'package:seyra/features/auth/presentation/pages/login_page.dart';

void main() {
  testWidgets('shows required-field validation on empty login submit', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: LoginPage(loginUseCase: LoginUseCase(_FakeAuthRepository())),
      ),
    );

    await tester.tap(find.byKey(const Key('login_submit_button')));
    await tester.pump();

    expect(find.text('Username is required'), findsOneWidget);
    expect(find.text('Password is required'), findsOneWidget);
  });

  testWidgets('shows unavailable message when authentication is not connected', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: LoginPage(loginUseCase: LoginUseCase(_FakeAuthRepository())),
      ),
    );

    await tester.enterText(
      find.byKey(const Key('login_username_field')),
      'ada',
    );
    await tester.enterText(
      find.byKey(const Key('login_password_field')),
      'secret',
    );
    await tester.tap(find.byKey(const Key('login_submit_button')));
    await tester.pump();

    expect(find.text('Authentication is not connected yet'), findsOneWidget);
    expect(find.text('ada'), findsOneWidget);
  });
}

final class _FakeAuthRepository implements AuthRepository {
  @override
  Future<Result<AuthSession>> login({
    required String username,
    required String password,
  }) async {
    return const FailureResult(AuthUnavailableFailure());
  }

  @override
  Future<Result<AuthSession>> register({
    required String username,
    required String password,
  }) {
    throw UnimplementedError();
  }

  @override
  Future<Result<void>> logout() {
    throw UnimplementedError();
  }

  @override
  Future<Result<AuthSession?>> restoreSession() {
    throw UnimplementedError();
  }

  @override
  Future<Result<AuthSession>> refreshSession() {
    throw UnimplementedError();
  }

  @override
  Future<Result<void>> deleteAccount({required String password}) {
    throw UnimplementedError();
  }
}
