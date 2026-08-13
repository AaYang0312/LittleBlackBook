package com.xbs.app.data.repository

import com.xbs.app.data.api.InteractionApi
import com.xbs.app.data.api.dataOrThrow
import com.xbs.app.data.api.dto.CommentDto
import com.xbs.app.data.api.dto.CommentReq
import com.xbs.app.data.api.successOrThrow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class InteractionRepository @Inject constructor(
    private val api: InteractionApi,
) {
    suspend fun setLiked(noteId: Long, liked: Boolean): Result<Unit> = runCatching {
        if (liked) api.like(noteId).successOrThrow() else api.unlike(noteId).successOrThrow()
    }

    suspend fun setCollected(noteId: Long, collected: Boolean): Result<Unit> = runCatching {
        if (collected) api.collect(noteId).successOrThrow() else api.uncollect(noteId).successOrThrow()
    }

    suspend fun setFollowed(userId: Long, followed: Boolean): Result<Unit> = runCatching {
        if (followed) api.follow(userId).successOrThrow() else api.unfollow(userId).successOrThrow()
    }

    suspend fun listComments(noteId: Long, cursor: Long?, size: Int = 20): Result<List<CommentDto>> = runCatching {
        api.listComments(noteId, cursor, size).dataOrThrow()
    }

    suspend fun createComment(noteId: Long, content: String): Result<CommentDto> = runCatching {
        api.createComment(noteId, CommentReq(content)).dataOrThrow()
    }
}
