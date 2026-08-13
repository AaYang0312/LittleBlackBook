package com.xbs.app.navigation

object Routes {
    const val LOGIN = "login"
    const val REGISTER = "register"
    const val DISCOVER = "discover"
    const val FOLLOWING = "following"
    const val PROFILE = "profile"
    const val PUBLISH = "publish"
    const val DETAIL = "detail/{noteId}"
    const val ARG_NOTE_ID = "noteId"
    fun detail(noteId: Long) = "detail/$noteId"

    val TAB_ROUTES = setOf(DISCOVER, FOLLOWING, PROFILE)
}
