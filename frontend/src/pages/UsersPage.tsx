import { useState, useEffect } from 'react';
import { Users, Plus, Trash2, Pencil, KeyRound } from 'lucide-react';
import { usersApi } from '@/api';
import { useAuth } from '../context/AuthContext';
import type { User, UserCreate } from '../types';

export default function UsersPage() {
  const { user: currentUser } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [editUser, setEditUser] = useState<User | null>(null);
  const [passwordUser, setPasswordUser] = useState<User | null>(null);

  // Create form state
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState<'admin' | 'reader'>('reader');

  // Edit form state
  const [editUsername, setEditUsername] = useState('');
  const [editRole, setEditRole] = useState<'admin' | 'reader'>('reader');

  // Password form state
  const [newPwd, setNewPwd] = useState('');

  useEffect(() => {
    fetchUsers();
  }, []);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const data = await usersApi.getAll();
      setUsers(data);
    } catch (err) {
      console.error('Failed to fetch users:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async () => {
    if (!newUsername || !newPassword) return;
    try {
      const data: UserCreate = { username: newUsername, password: newPassword, role: newRole };
      await usersApi.create(data);
      setNewUsername('');
      setNewPassword('');
      setNewRole('reader');
      setShowCreate(false);
      fetchUsers();
    } catch (err) {
      console.error('Failed to create user:', err);
      alert('Failed to create user');
    }
  };

  const handleUpdate = async () => {
    if (!editUser) return;
    try {
      await usersApi.update(editUser.id, { username: editUsername, role: editRole });
      setEditUser(null);
      fetchUsers();
    } catch (err) {
      console.error('Failed to update user:', err);
      alert('Failed to update user');
    }
  };

  const handleChangePassword = async () => {
    if (!passwordUser || !newPwd) return;
    try {
      await usersApi.changePassword(passwordUser.id, newPwd);
      setPasswordUser(null);
      setNewPwd('');
      alert('Password updated successfully');
    } catch (err) {
      console.error('Failed to change password:', err);
      alert('Failed to change password');
    }
  };

  const handleDelete = async (user: User) => {
    if (!confirm(`Delete user "${user.username}"? This cannot be undone.`)) return;
    try {
      await usersApi.delete(user.id);
      fetchUsers();
    } catch (err) {
      console.error('Failed to delete user:', err);
      alert('Failed to delete user');
    }
  };

  const roleBadge = (role: string) => (
    <span className={`px-2 py-0.5 text-xs rounded font-medium ${
      role === 'admin'
        ? 'bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300'
        : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
    }`}>
      {role}
    </span>
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Users</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Manage user accounts. Only admins can access this page.
        </p>
      </div>

      <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Users className="h-5 w-5 text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">All Users</h2>
          </div>
          {!showCreate && (
            <button
              onClick={() => setShowCreate(true)}
              className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700"
            >
              <Plus className="h-4 w-4" />
              Create User
            </button>
          )}
        </div>

        {/* Create form */}
        {showCreate && (
          <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 mb-4">
            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">New User</h3>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Username</label>
                <input
                  type="text"
                  value={newUsername}
                  onChange={(e) => setNewUsername(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Password</label>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Role</label>
                <select
                  value={newRole}
                  onChange={(e) => setNewRole(e.target.value as 'admin' | 'reader')}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                >
                  <option value="reader">reader</option>
                  <option value="admin">admin</option>
                </select>
              </div>
            </div>
            <div className="flex gap-2 mt-3">
              <button
                onClick={handleCreate}
                disabled={!newUsername || !newPassword}
                className="px-4 py-2 bg-indigo-600 text-white text-sm rounded hover:bg-indigo-700 disabled:opacity-50"
              >
                Create
              </button>
              <button
                onClick={() => { setShowCreate(false); setNewUsername(''); setNewPassword(''); }}
                className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm rounded hover:bg-gray-300 dark:hover:bg-gray-600"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Edit form */}
        {editUser && (
          <div className="border border-indigo-200 dark:border-indigo-700 rounded-lg p-4 mb-4 bg-indigo-50 dark:bg-indigo-900/10">
            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">Edit User: {editUser.username}</h3>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Username</label>
                <input
                  type="text"
                  value={editUsername}
                  onChange={(e) => setEditUsername(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Role</label>
                <select
                  value={editRole}
                  onChange={(e) => setEditRole(e.target.value as 'admin' | 'reader')}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                >
                  <option value="reader">reader</option>
                  <option value="admin">admin</option>
                </select>
              </div>
            </div>
            <div className="flex gap-2 mt-3">
              <button
                onClick={handleUpdate}
                className="px-4 py-2 bg-indigo-600 text-white text-sm rounded hover:bg-indigo-700"
              >
                Save
              </button>
              <button
                onClick={() => setEditUser(null)}
                className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm rounded hover:bg-gray-300 dark:hover:bg-gray-600"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Change password form */}
        {passwordUser && (
          <div className="border border-yellow-200 dark:border-yellow-700 rounded-lg p-4 mb-4 bg-yellow-50 dark:bg-yellow-900/10">
            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">Change password for: {passwordUser.username}</h3>
            <div className="flex gap-3 items-end">
              <div className="flex-1">
                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">New Password</label>
                <input
                  type="password"
                  value={newPwd}
                  onChange={(e) => setNewPwd(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
              <button
                onClick={handleChangePassword}
                disabled={!newPwd}
                className="px-4 py-2 bg-yellow-600 text-white text-sm rounded hover:bg-yellow-700 disabled:opacity-50"
              >
                Change
              </button>
              <button
                onClick={() => { setPasswordUser(null); setNewPwd(''); }}
                className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm rounded hover:bg-gray-300 dark:hover:bg-gray-600"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Users list */}
        {loading ? (
          <div className="flex justify-center py-8">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-indigo-600"></div>
          </div>
        ) : (
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {users.map((u) => (
              <div key={u.id} className="flex items-center justify-between py-3">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-full bg-indigo-100 dark:bg-indigo-900/40 flex items-center justify-center">
                    <span className="text-indigo-600 dark:text-indigo-300 text-sm font-medium">
                      {u.username[0].toUpperCase()}
                    </span>
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900 dark:text-white text-sm">{u.username}</span>
                      {roleBadge(u.role)}
                      {u.id === currentUser?.id && (
                        <span className="text-xs text-gray-400">(you)</span>
                      )}
                    </div>
                    <div className="text-xs text-gray-400">
                      Created {new Date(u.created_at).toLocaleDateString()}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => { setEditUser(u); setEditUsername(u.username); setEditRole(u.role); setPasswordUser(null); }}
                    className="p-2 text-gray-500 hover:text-indigo-600 hover:bg-indigo-50 dark:hover:bg-indigo-900/20 rounded"
                    title="Edit user"
                  >
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => { setPasswordUser(u); setEditUser(null); setNewPwd(''); }}
                    className="p-2 text-gray-500 hover:text-yellow-600 hover:bg-yellow-50 dark:hover:bg-yellow-900/20 rounded"
                    title="Change password"
                  >
                    <KeyRound className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(u)}
                    disabled={u.id === currentUser?.id}
                    className="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded disabled:opacity-30 disabled:cursor-not-allowed"
                    title="Delete user"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
