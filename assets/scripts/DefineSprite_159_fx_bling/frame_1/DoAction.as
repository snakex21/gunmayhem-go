_X = asdf._x;
_Y = asdf._y;
_alpha = 0;
time = 0;
_xscale = 700;
_yscale = _xscale;
yoffset = 30;
if(_X == 0 && _Y == 0)
{
   removeMovieClip(this);
   delete this.onEnterFrame;
}
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _X = asdf._x;
      _Y = asdf._y - yoffset;
      time += 1;
      if(time <= 20)
      {
         if(_alpha < 100)
         {
            _alpha = _alpha + 50;
         }
         if(_xscale > 100)
         {
            _xscale = _xscale + (90 - _xscale) / 5;
         }
         _yscale = _xscale;
      }
      else if(time >= 40)
      {
         _alpha = _alpha - 10;
         if(_alpha <= 1)
         {
            removeMovieClip(this);
            delete this.onEnterFrame;
         }
      }
      yoffset += (110 - yoffset) / 5;
      if(_root.deleteeverything || _root.gamewin)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
gotoAndStop(mod + 1);
if(_root.gamemode == 4)
{
   levelnumber.text = leveldisplay;
}
score = 0;
switch(mod)
{
   case 1:
      score = 100;
      break;
   case 2:
      score = 200;
      break;
   case 3:
      score = 300;
      break;
   case 4:
      score = 50;
}
_root.pgsdata[asdf.PLAYERNUMBER - 1][7] += score;
