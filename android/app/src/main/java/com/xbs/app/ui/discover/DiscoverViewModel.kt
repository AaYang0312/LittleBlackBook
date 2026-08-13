package com.xbs.app.ui.discover

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.xbs.app.data.api.dto.NoteDto
import com.xbs.app.data.repository.NoteRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class DiscoverUiState(
    val items: List<NoteDto> = emptyList(),
    val isRefreshing: Boolean = false,
    val isLoadingMore: Boolean = false,
    val hasMore: Boolean = true,
    val loadMoreError: Boolean = false,
    val nextCursor: Long = 0,
)

@HiltViewModel
class DiscoverViewModel @Inject constructor(
    private val repo: NoteRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(DiscoverUiState())
    val uiState: StateFlow<DiscoverUiState> = _uiState.asStateFlow()

    init { refresh() }

    fun refresh() {
        if (_uiState.value.isRefreshing) return
        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true, loadMoreError = false) }
            repo.latest(cursor = null)
                .onSuccess { page ->
                    _uiState.update {
                        it.copy(items = page.list, nextCursor = page.nextCursor, hasMore = page.hasMore)
                    }
                }
                .onFailure { _uiState.update { it.copy(loadMoreError = it.items.isNotEmpty()) } }
            _uiState.update { it.copy(isRefreshing = false) }
        }
    }

    fun loadMore() {
        val s = _uiState.value
        if (s.isRefreshing || s.isLoadingMore || !s.hasMore) return  // 防重入
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingMore = true, loadMoreError = false) }
            repo.latest(cursor = s.nextCursor)
                .onSuccess { page ->
                    _uiState.update {
                        it.copy(
                            items = it.items + page.list,
                            nextCursor = page.nextCursor,
                            hasMore = page.hasMore,
                        )
                    }
                }
                .onFailure { _uiState.update { it.copy(loadMoreError = true) } }
            _uiState.update { it.copy(isLoadingMore = false) }
        }
    }
}
