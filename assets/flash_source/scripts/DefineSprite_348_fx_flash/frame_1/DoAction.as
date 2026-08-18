if(random(2) == 0)
{
   _yscale = _yscale * -1;
}
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _alpha = _alpha - 51;
      if(_alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
