import { NavLink, useNavigate } from 'react-router-dom';
import { Boxes, Package, FolderTree, Key, Sun, Moon, FileJson, Users, LogOut } from 'lucide-react';
import { useTheme } from '../context/ThemeContext';
import { useAuth } from '../context/AuthContext';

interface LayoutProps {
  children: React.ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  const { isDark, toggleTheme } = useTheme();
  const { user, logout, isAdmin } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login', { replace: true });
  };

  const navItems = [
    { to: '/modules', label: 'Modules', icon: Boxes },
    { to: '/providers', label: 'Providers', icon: Package },
    { to: '/namespaces', label: 'Namespaces', icon: FolderTree },
    { to: '/api-keys', label: 'API Keys', icon: Key },
    { to: '/tfstate', label: 'TF State', icon: FileJson },
  ];

  if (isAdmin) {
    navItems.push({ to: '/users', label: 'Users', icon: Users });
  }

  return (
    <div className="min-h-screen flex bg-gray-50 dark:bg-gray-950">
      {/* Sidebar */}
      <aside className="w-64 bg-gray-900 text-white flex flex-col">
        <div className="p-6 flex items-center justify-between">
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Boxes className="w-8 h-8" />
            Registry
          </h1>
          <button
            onClick={toggleTheme}
            className="p-2 rounded-lg hover:bg-gray-800 transition-colors text-gray-400 hover:text-white"
            title={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            {isDark ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
          </button>
        </div>
        <nav className="mt-6 flex-1">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-6 py-3 text-sm font-medium transition-colors ${isActive
                  ? 'bg-gray-800 text-white border-l-4 border-blue-500'
                  : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`
              }
            >
              <item.icon className="w-5 h-5" />
              {item.label}
            </NavLink>
          ))}
        </nav>

        {/* User info + logout */}
        <div className="p-4 border-t border-gray-800">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 min-w-0">
              <div className="w-7 h-7 rounded-full bg-indigo-700 flex items-center justify-center flex-shrink-0">
                <span className="text-xs font-medium text-white">
                  {user?.username?.[0]?.toUpperCase()}
                </span>
              </div>
              <div className="min-w-0">
                <div className="text-sm font-medium text-white truncate">{user?.username}</div>
                <div className="text-xs text-gray-400">{user?.role}</div>
              </div>
            </div>
            <button
              onClick={handleLogout}
              className="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded transition-colors"
              title="Sign out"
            >
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto bg-gray-50 dark:bg-gray-900">
        <div className="p-8">{children}</div>
      </main>
    </div>
  );
}
