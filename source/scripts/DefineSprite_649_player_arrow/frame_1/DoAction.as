_X = 480;
_Y = 300;
_alpha = 0;
switch(this._name)
{
   case "arrow1":
      target = _root.player1;
      break;
   case "arrow2":
      target = _root.player2;
      break;
   case "arrow3":
      target = _root.player3;
      break;
   case "arrow4":
      target = _root.player4;
}
arrow.gotoAndStop(target.playernumber + 1);
this.onEnterFrame = function()
{
   radians = Math.atan2(target._y - _Y,target._x - _X);
   degrees = radians * 180 / 3.141592653589793;
   _X = - _root._x + 450;
   _Y = - _root._y + 300;
   _rotation = degrees + 90;
   dist._rotation = - _rotation;
   dist.disttext.text = Math.round(Math.sqrt(Math.pow(target._x - _X,2) + Math.pow(target._y - _Y,2)));
   if(target._y <= - _root._y + 50 || target._y >= - _root._y + 550 || target._x <= - _root._x + 50 || target._x >= - _root._x + 910)
   {
      _alpha = 100;
   }
   else
   {
      _alpha = 0;
   }
   if(target._alpha != 100 && target.lives <= 0)
   {
      removeMovieClip(this);
      delete this.onEnterFrame;
   }
};
