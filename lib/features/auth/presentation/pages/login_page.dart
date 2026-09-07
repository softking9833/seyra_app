import 'package:flutter/material.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/features/auth/domain/entities/auth_session.dart';
import 'package:seyra/features/auth/domain/usecases/login_use_case.dart';
import 'package:seyra/features/auth/presentation/widgets/auth_screen_scaffold.dart';
import 'package:seyra/features/auth/presentation/widgets/auth_text_field.dart';
import 'package:seyra/features/auth/presentation/widgets/seyra_auth_header.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({
    super.key,
    required this.loginUseCase,
    this.onCreateAccount,
    this.onAuthenticated,
  });

  final LoginUseCase loginUseCase;
  final VoidCallback? onCreateAccount;
  final ValueChanged<AuthSession>? onAuthenticated;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();

  bool _submitting = false;
  String? _errorMessage;

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() => _errorMessage = null);
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }

    setState(() => _submitting = true);
    final result = await widget.loginUseCase(
      username: _usernameController.text,
      password: _passwordController.text,
    );
    if (!mounted) {
      return;
    }

    switch (result) {
      case FailureResult(:final failure):
        setState(() {
          _submitting = false;
          _errorMessage = failure.message;
        });
      case Success(:final value):
        setState(() => _submitting = false);
        widget.onAuthenticated?.call(value);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final canPop = Navigator.of(context).canPop();

    return AuthScreenScaffold(
      showBack: canPop,
      child: Form(
        key: _formKey,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const SeyraAuthHeader(
              subtitle: 'Your privacy matters. Sign in to continue.',
            ),
            const SizedBox(height: 32),
            AuthTextField(
              key: const Key('login_username_field'),
              controller: _usernameController,
              label: 'Username',
              prefixIcon: Icons.person_outline,
              enabled: !_submitting,
              textInputAction: TextInputAction.next,
              autofillHints: const [AutofillHints.username],
              validator: (value) {
                if (value == null || value.trim().isEmpty) {
                  return 'Username is required';
                }
                return null;
              },
            ),
            const SizedBox(height: 14),
            AuthTextField(
              key: const Key('login_password_field'),
              controller: _passwordController,
              label: 'Password',
              prefixIcon: Icons.lock_outline,
              obscureable: true,
              enabled: !_submitting,
              textInputAction: TextInputAction.done,
              autofillHints: const [AutofillHints.password],
              validator: (value) {
                if (value == null || value.isEmpty) {
                  return 'Password is required';
                }
                return null;
              },
            ),
            if (_errorMessage != null) ...[
              const SizedBox(height: 16),
              Text(
                _errorMessage!,
                key: const Key('login_error_message'),
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ],
            const SizedBox(height: 24),
            FilledButton(
              key: const Key('login_submit_button'),
              onPressed: _submitting ? null : _submit,
              child: _submitting
                  ? SizedBox(
                      height: 22,
                      width: 22,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: theme.colorScheme.onPrimary,
                      ),
                    )
                  : const Text('Sign In'),
            ),
            TextButton(
              onPressed: _submitting
                  ? null
                  : () {
                      // Visual placeholder. Recovery is not implemented yet.
                    },
              child: const Text('Forgot password?'),
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                const Expanded(child: Divider()),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                  child: Text('or', style: theme.textTheme.bodyMedium),
                ),
                const Expanded(child: Divider()),
              ],
            ),
            const SizedBox(height: 16),
            OutlinedButton(
              onPressed: _submitting ? null : widget.onCreateAccount,
              child: const Text('Create an account'),
            ),
          ],
        ),
      ),
    );
  }
}
