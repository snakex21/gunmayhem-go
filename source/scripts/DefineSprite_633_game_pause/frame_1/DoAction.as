_X = _root.hud._x;
_Y = _root.hud._y;
_root.GAMEPAUSED = true;
btn1.onRelease = function()
{
   _root.GAMEPAUSED = false;
   removeMovieClip(_root.game_pause);
   delete _root.game_pause.onEnterFrame;
};
btn2.onRelease = function()
{
   _root.GAMEPAUSED = false;
   _root.endgame();
};
