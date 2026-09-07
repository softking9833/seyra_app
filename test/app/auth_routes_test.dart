import 'package:flutter_test/flutter_test.dart';
import 'package:seyra/app/app.dart';
import 'package:seyra/app/di/app_dependencies.dart';
import 'package:seyra/features/auth/presentation/pages/login_page.dart';
import 'package:seyra/features/auth/presentation/pages/register_page.dart';

void main() {
  setUp(() async {
    await AppDependencies.initialize();
  });

  testWidgets('navigates from the shell to the login page', (tester) async {
    await tester.pumpWidget(const SeyraApp());

    await tester.tap(find.text('Sign in'));
    await tester.pumpAndSettle();

    expect(find.byType(LoginPage), findsOneWidget);
  });

  testWidgets('navigates from the shell to the registration page', (
    tester,
  ) async {
    await tester.pumpWidget(const SeyraApp());

    await tester.tap(find.text('Create an account'));
    await tester.pumpAndSettle();

    expect(find.byType(RegisterPage), findsOneWidget);
  });

  testWidgets('navigates from login to registration', (tester) async {
    await tester.pumpWidget(const SeyraApp());

    await tester.tap(find.text('Sign in'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create an account'));
    await tester.pumpAndSettle();

    expect(find.byType(RegisterPage), findsOneWidget);
  });
}
