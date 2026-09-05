"""Polling regressions independent of a Windows console installation."""
import unittest
from unittest.mock import Mock, patch

from native_check import WindowsTerminal


class NativePollingTests(unittest.TestCase):
    def terminal(self):
        terminal = WindowsTerminal.__new__(WindowsTerminal)
        terminal.output = bytearray(b'last frame')
        terminal.pump = Mock()
        return terminal

    def test_transient_replace_read_error_is_retried(self):
        terminal = self.terminal()
        predicate = Mock(side_effect=[PermissionError('replace in progress'), False, True])
        with patch('native_check.time.monotonic', side_effect=[0, 1, 2, 3]):
            terminal.until(predicate, 'save')
        self.assertEqual(predicate.call_count, 3)

    def test_persistent_read_error_still_fails_by_original_deadline(self):
        terminal = self.terminal()
        predicate = Mock(side_effect=PermissionError('denied'))
        with patch('native_check.time.monotonic', side_effect=[0, 1, 14, 15]):
            with self.assertRaisesRegex(AssertionError, 'save: last read error=.*denied.*last frame'):
                terminal.until(predicate, 'save')
        self.assertEqual(predicate.call_count, 2)

    def test_unrelated_predicate_error_is_not_swallowed(self):
        with self.assertRaisesRegex(ValueError, 'broken assertion'):
            self.terminal().until(Mock(side_effect=ValueError('broken assertion')), 'save')


if __name__ == '__main__':
    unittest.main()
