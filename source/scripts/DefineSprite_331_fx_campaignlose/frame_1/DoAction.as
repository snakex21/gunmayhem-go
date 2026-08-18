this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(_root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
      _X = - _root._x;
      _Y = - _root._y;
   }
};
