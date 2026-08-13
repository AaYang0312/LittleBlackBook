package com.xbs.app.ui.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.xbs.app.data.repository.UserRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class AuthUiState(
    val isLoading: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class AuthViewModel @Inject constructor(
    private val repo: UserRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AuthUiState())
    val uiState: StateFlow<AuthUiState> = _uiState.asStateFlow()

    /** 登录成功（含注册后自动登录）事件，页面收到后跳转主页。 */
    private val _loggedIn = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val loggedIn: SharedFlow<Unit> = _loggedIn.asSharedFlow()

    fun login(username: String, password: String) {
        if (username.isBlank() || password.isBlank()) {
            _uiState.update { it.copy(error = "用户名和密码不能为空") }
            return
        }
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            repo.login(username.trim(), password)
                .onSuccess { _loggedIn.emit(Unit) }
                .onFailure { e -> _uiState.update { it.copy(error = e.message ?: "登录失败") } }
            _uiState.update { it.copy(isLoading = false) }
        }
    }

    fun register(username: String, password: String, nickname: String) {
        if (username.isBlank() || password.length < 6) {
            _uiState.update { it.copy(error = "用户名必填，密码至少 6 位") }
            return
        }
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            repo.register(username.trim(), password, nickname.trim())
                .onSuccess { login(username.trim(), password) }  // 注册成功自动登录
                .onFailure { e -> _uiState.update { it.copy(error = e.message ?: "注册失败") } }
            _uiState.update { it.copy(isLoading = false) }
        }
    }
}
