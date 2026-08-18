stop();
_xscale = 80 * asdf;
_yscale = 80;
if(_rotation < 0)
{
   _xscale = 50 * asdf;
   _yscale = 50;
   _rotation = Math.abs(_rotation);
}
PLAYERNUMBER = Math.floor(_rotation);
switch(PLAYERNUMBER)
{
   case 1:
      playernumber = _root.p1color;
      break;
   case 2:
      playernumber = _root.p2color;
      break;
   case 3:
      playernumber = _root.p3color;
      break;
   case 4:
      playernumber = _root.p4color;
}
_rotation = 0;
_alpha = 20;
time = 10;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      time -= 1;
      if(_alpha <= 1 || time <= 0)
      {
         this.swapDepths(1);
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
