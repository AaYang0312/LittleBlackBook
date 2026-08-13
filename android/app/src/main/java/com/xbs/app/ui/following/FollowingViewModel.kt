package com.xbs.app.ui.following

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.xbs.app.data.api.dto.NoteDto
import com.xbs.app.data.repository.FeedRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class FollowingUiState(
    val items: List<NoteDto> = emptyList(),
    val isRefreshing: Boolean = false,
    val isLoadingMore: Boolean = false,
    val hasMore: Boolean = true,
    val loadMoreError: Boolean = false,
)

private const val PAGE_SIZE = 20

@HiltViewModel
class FollowingViewModel @Inject constructor(
    private val repo: FeedRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(FollowingUiState())
    val uiState: StateFlow<FollowingUiState> = _uiState.asStateFlow()

    init { refresh() }

    fun refresh() {
        if (_uiState.value.isRefreshing) return
        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true, loadMoreError = false) }
            repo.following(offset = 0, size = PAGE_SIZE)
                .onSuccess { list ->
                    _uiState.update { it.copy(items = list, hasMore = list.size >= PAGE_SIZE) }
                }
                .onFailure { _uiState.update { it.copy(loadMoreError = it.items.isNotEmpty()) } }
            _uiState.update { it.copy(isRefreshing = false) }
        }
    }

    fun loadMore() {
        val s = _uiState.value
        if (s.isRefreshing || s.isLoadingMore || !s.hasMore) return
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingMore = true, loadMoreError = false) }
            repo.following(offset = s.items.size, size = PAGE_SIZE)
                .onSuccess { list ->
                    _uiState.update {
                        it.copy(items = it.items + list, hasMore = list.size >= PAGE_SIZE)
                    }
                }
                .onFailure { _uiState.update { it.copy(loadMoreError = true) } }
            _uiState.update { it.copy(isLoadingMore = false) }
        }
    }
}
