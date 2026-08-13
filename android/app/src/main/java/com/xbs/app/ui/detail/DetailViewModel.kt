package com.xbs.app.ui.detail

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.xbs.app.data.api.dto.CommentDto
import com.xbs.app.data.api.dto.NoteDto
import com.xbs.app.data.repository.InteractionRepository
import com.xbs.app.data.repository.NoteRepository
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

data class DetailUiState(
    val note: NoteDto? = null,
    val isLoading: Boolean = false,
    val error: String? = null,
    // 后端无 liked_by_me 字段：初始一律未点赞/未收藏，点击后乐观翻转（已知取舍）
    val liked: Boolean = false,
    val likeCount: Long = 0,
    val collected: Boolean = false,
    val collectCount: Long = 0,
    val followed: Boolean = false,
    val comments: List<CommentDto> = emptyList(),
    val commentsHasMore: Boolean = true,
    val isLoadingComments: Boolean = false,
    val commentCount: Long = 0,
)

private const val COMMENT_PAGE_SIZE = 20

@HiltViewModel
class DetailViewModel @Inject constructor(
    private val savedStateHandle: androidx.lifecycle.SavedStateHandle,
    private val noteRepo: NoteRepository,
    private val interactionRepo: InteractionRepository,
) : ViewModel() {

    constructor(noteId: Long, noteRepo: NoteRepository, interactionRepo: InteractionRepository) :
        this(
            androidx.lifecycle.SavedStateHandle(mapOf("noteId" to noteId)),
            noteRepo,
            interactionRepo,
        )

    private val noteId: Long = checkNotNull(savedStateHandle["noteId"])

    private val _uiState = MutableStateFlow(DetailUiState())
    val uiState: StateFlow<DetailUiState> = _uiState.asStateFlow()

    private val _toasts = MutableSharedFlow<String>(extraBufferCapacity = 1)
    val toasts: SharedFlow<String> = _toasts.asSharedFlow()

    fun load() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            noteRepo.detail(noteId)
                .onSuccess { n ->
                    _uiState.update {
                        it.copy(
                            note = n,
                            likeCount = n.likeCount,
                            collectCount = n.collectCount,
                            commentCount = n.commentCount,
                        )
                    }
                }
                .onFailure { e -> _uiState.update { it.copy(error = e.message ?: "加载失败") } }
            _uiState.update { it.copy(isLoading = false) }
            refreshComments()
        }
    }

    fun toggleLike() {
        val target = !_uiState.value.liked
        _uiState.update { it.copy(liked = target, likeCount = it.likeCount + if (target) 1 else -1) }
        viewModelScope.launch {
            interactionRepo.setLiked(noteId, target).onFailure {
                _uiState.update { it.copy(liked = !target, likeCount = it.likeCount + if (target) -1 else 1) }
                _toasts.emit("操作失败，请重试")
            }
        }
    }

    fun toggleCollect() {
        val target = !_uiState.value.collected
        _uiState.update { it.copy(collected = target, collectCount = it.collectCount + if (target) 1 else -1) }
        viewModelScope.launch {
            interactionRepo.setCollected(noteId, target).onFailure {
                _uiState.update { it.copy(collected = !target, collectCount = it.collectCount + if (target) -1 else 1) }
                _toasts.emit("操作失败，请重试")
            }
        }
    }

    fun toggleFollow() {
        val authorId = _uiState.value.note?.userId ?: return
        val target = !_uiState.value.followed
        _uiState.update { it.copy(followed = target) }
        viewModelScope.launch {
            interactionRepo.setFollowed(authorId, target).onFailure {
                _uiState.update { it.copy(followed = !target) }
                _toasts.emit("操作失败，请重试")
            }
        }
    }

    private fun refreshComments() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingComments = true) }
            interactionRepo.listComments(noteId, cursor = null)
                .onSuccess { list ->
                    _uiState.update { it.copy(comments = list, commentsHasMore = list.size >= COMMENT_PAGE_SIZE) }
                }
            _uiState.update { it.copy(isLoadingComments = false) }
        }
    }

    fun loadMoreComments() {
        val s = _uiState.value
        if (s.isLoadingComments || !s.commentsHasMore) return
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingComments = true) }
            interactionRepo.listComments(noteId, cursor = s.comments.lastOrNull()?.id)
                .onSuccess { list ->
                    _uiState.update { it.copy(comments = it.comments + list, commentsHasMore = list.size >= COMMENT_PAGE_SIZE) }
                }
            _uiState.update { it.copy(isLoadingComments = false) }
        }
    }

    fun sendComment(content: String) {
        if (content.isBlank()) return
        viewModelScope.launch {
            interactionRepo.createComment(noteId, content.trim())
                .onSuccess { c ->
                    _uiState.update { it.copy(comments = listOf(c) + it.comments, commentCount = it.commentCount + 1) }
                }
                .onFailure { _toasts.emit("评论失败，请重试") }
        }
    }
}
