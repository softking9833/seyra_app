import 'package:flutter/material.dart';
import 'package:seyra/core/errors/result.dart';
import 'package:seyra/core/theme/app_colors.dart';
import 'package:seyra/features/auth/domain/entities/user.dart';
import 'package:seyra/features/profile/domain/usecases/update_profile_use_case.dart';
import 'package:seyra/features/profile/domain/usecases/watch_profile_use_case.dart';

class EditProfilePage extends StatefulWidget {
  const EditProfilePage({
    super.key,
    required this.user,
    required this.watchProfile,
    required this.updateProfile,
  });

  final User user;
  final WatchProfileUseCase watchProfile;
  final UpdateProfileUseCase updateProfile;

  @override
  State<EditProfilePage> createState() => _EditProfilePageState();
}

class _EditProfilePageState extends State<EditProfilePage> {
  final _displayName = TextEditingController();
  final _bio = TextEditingController();
  bool _loaded = false;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    widget
        .watchProfile(userId: widget.user.id, username: widget.user.username)
        .first
        .then((profile) {
          if (!mounted) {
            return;
          }
          _displayName.text = profile.displayName;
          _bio.text = profile.bio;
          setState(() => _loaded = true);
        });
  }

  @override
  void dispose() {
    _displayName.dispose();
    _bio.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    setState(() {
      _saving = true;
      _error = null;
    });
    final result = await widget.updateProfile(
      userId: widget.user.id,
      displayName: _displayName.text,
      bio: _bio.text,
    );
    if (!mounted) {
      return;
    }
    switch (result) {
      case FailureResult(:final failure):
        setState(() {
          _saving = false;
          _error = failure.message;
        });
      case Success():
        Navigator.of(context).pop();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.surfaceMuted,
      appBar: AppBar(
        title: const Text('Edit profile'),
        actions: [
          TextButton(
            key: const Key('edit_profile_save_button'),
            onPressed: _loaded && !_saving ? _save : null,
            child: const Text('Save'),
          ),
        ],
      ),
      body: !_loaded
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(20),
              children: [
                TextField(
                  key: const Key('edit_profile_display_name_field'),
                  controller: _displayName,
                  textCapitalization: TextCapitalization.words,
                  decoration: const InputDecoration(
                    labelText: 'Display name',
                  ),
                ),
                const SizedBox(height: 16),
                TextField(
                  key: const Key('edit_profile_bio_field'),
                  controller: _bio,
                  maxLines: 3,
                  decoration: const InputDecoration(
                    labelText: 'Bio',
                    alignLabelWithHint: true,
                  ),
                ),
                if (_error != null) ...[
                  const SizedBox(height: 12),
                  Text(
                    _error!,
                    style: const TextStyle(color: Color(0xFFB91C1C)),
                  ),
                ],
                const SizedBox(height: 12),
                const Text(
                  'Your username stays the same. Display name and bio are stored locally for now.',
                  style: TextStyle(color: AppColors.textSecondary),
                ),
              ],
            ),
    );
  }
}
