import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:seyra/app/app.dart';
import 'package:seyra/app/di/app_dependencies.dart';
import 'package:seyra/features/auth/data/datasources/mock_auth_remote_data_source.dart';
import 'package:seyra/features/profile/presentation/pages/edit_profile_page.dart';
import 'package:seyra/features/profile/presentation/pages/settings_page.dart';

void main() {
  setUp(() async {
    await AppDependencies.initialize(
      remoteDataSource: MockAuthRemoteDataSource(),
    );
  });

  Future<void> register(WidgetTester tester) async {
    await tester.pumpWidget(const SeyraApp());
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create an account'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('register_username_field')),
      'ada',
    );
    await tester.enterText(
      find.byKey(const Key('register_password_field')),
      'secret',
    );
    await tester.enterText(
      find.byKey(const Key('register_confirm_password_field')),
      'secret',
    );
    await tester.tap(find.byKey(const Key('register_submit_button')));
    await tester.pumpAndSettle();
  }

  testWidgets('opens profile and settings, then logs out', (tester) async {
    await register(tester);

    await tester.tap(
      find.descendant(
        of: find.byType(NavigationBar),
        matching: find.text('Profile'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Ada'), findsWidgets);
    expect(find.text('@ada'), findsOneWidget);
    expect(find.text('Edit Profile'), findsOneWidget);

    await tester.tap(find.byKey(const Key('profile_open_settings_button')));
    await tester.pumpAndSettle();
    expect(find.byType(SettingsPage), findsOneWidget);

    await tester.scrollUntilVisible(
      find.byKey(const Key('settings_logout_tile')),
      400,
    );
    await tester.tap(find.byKey(const Key('settings_logout_tile')));
    await tester.pumpAndSettle();

    expect(find.text('Sign in'), findsOneWidget);
    expect(find.byType(SettingsPage), findsNothing);
  });

  testWidgets('delete account confirmation does not sign the user out', (
    tester,
  ) async {
    await register(tester);
    await tester.tap(
      find.descendant(
        of: find.byType(NavigationBar),
        matching: find.text('Profile'),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('profile_open_settings_button')));
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.byKey(const Key('settings_delete_account_tile')),
      400,
    );
    await tester.tap(find.byKey(const Key('settings_delete_account_tile')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('delete_account_confirm_button')));
    await tester.pumpAndSettle();

    expect(find.text('Account deletion is not available yet'), findsOneWidget);
    expect(find.byType(SettingsPage), findsOneWidget);
    expect(find.text('Sign in'), findsNothing);
  });

  testWidgets('edit profile saves a local display name', (tester) async {
    await register(tester);
    await tester.tap(
      find.descendant(
        of: find.byType(NavigationBar),
        matching: find.text('Profile'),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('edit_profile_button')));
    await tester.pumpAndSettle();

    expect(find.byType(EditProfilePage), findsOneWidget);
    await tester.enterText(
      find.byKey(const Key('edit_profile_display_name_field')),
      'Ada Lovelace',
    );
    await tester.tap(find.byKey(const Key('edit_profile_save_button')));
    await tester.pumpAndSettle();

    expect(find.byType(EditProfilePage), findsNothing);
    expect(find.text('Ada Lovelace'), findsOneWidget);
  });
}
